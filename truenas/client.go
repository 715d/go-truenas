package truenas

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/gorilla/websocket"
	"github.com/puzpuzpuz/xsync/v3"
)

type Options struct {
	Username            string
	Password            string
	APIKey              string
	Debug               bool
	DefaultWriteTimeout time.Duration
	NotificationHandler func(method string, params json.RawMessage)
}

type Client struct {
	// Type-safe API clients
	Auth         *AuthClient
	Pool         *PoolClient
	Dataset      *DatasetClient
	Service      *ServiceClient
	System       *SystemClient
	Network      *NetworkClient
	SMB          *SMBClient
	NFS          *NFSClient
	SSH          *SSHClient
	Smart        *SmartClient
	VM           *VMClient
	Job          *JobClient
	VMDevice     *VMDeviceClient
	User         *UserClient
	Group        *GroupClient
	Alert        *AlertClient
	AlertService *AlertServiceClient
	Boot         *BootClient
	Certificate  *CertificateClient
	Cronjob      *CronjobClient
	Disk         *DiskClient
	APIKey       *APIKeyClient
	Filesystem   *FilesystemClient
	Sharing      *SharingClient

	// Internal state
	url         string
	conn        *websocket.Conn
	opts        Options
	mu          sync.RWMutex
	msgID       atomic.Int64
	pending     *xsync.MapOf[int64, chan Response]
	writeChan   chan *Request
	errCh       chan error
	reconnectCh chan struct{}
	doneCh      chan struct{} // Signal when client should shut down
	closed      atomic.Bool
	wg          sync.WaitGroup
}

// NewClient builds a new TrueNAS Client.
// Close() should be called to clean up resources when the client is no longer needed.
func NewClient(endpoint string, opts Options) (*Client, error) {
	c := &Client{
		url:         endpoint,
		opts:        opts,
		pending:     xsync.NewMapOf[int64, chan Response](),
		errCh:       make(chan error, 1),
		reconnectCh: make(chan struct{}, 1),
		doneCh:      make(chan struct{}),
	}
	if c.opts.DefaultWriteTimeout == 0 {
		c.opts.DefaultWriteTimeout = 5 * time.Second
	}

	// Initialize type-safe API clients
	c.Auth = NewAuthClient(c)
	c.Pool = NewPoolClient(c)
	c.Dataset = NewDatasetClient(c)
	c.Service = NewServiceClient(c)
	c.System = NewSystemClient(c)
	c.Network = NewNetworkClient(c)
	c.SMB = NewSMBClient(c)
	c.NFS = NewNFSClient(c)
	c.SSH = NewSSHClient(c)
	c.Smart = NewSmartClient(c)
	c.VM = NewVMClient(c)
	c.VMDevice = NewVMDeviceClient(c)
	c.User = NewUserClient(c)
	c.Group = NewGroupClient(c)
	c.Alert = NewAlertClient(c)
	c.Job = NewJobClient(c)
	c.AlertService = NewAlertServiceClient(c)
	c.Boot = NewBootClient(c)
	c.Certificate = NewCertificateClient(c)
	c.Cronjob = NewCronjobClient(c)
	c.Disk = NewDiskClient(c)
	c.APIKey = NewAPIKeyClient(c)
	c.Filesystem = NewFilesystemClient(c)
	c.Sharing = NewSharingClient(c)

	if err := c.connect(); err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	c.wg.Add(1)
	go c.connectionManager()

	if err := c.authenticate(); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("authentication: %w", err)
	}

	return c, nil
}

func (c *Client) Close() error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil // Already closed
	}

	// Cancel all pending requests by closing their channels.
	c.pending.Range(func(id int64, ch chan Response) bool {
		close(ch)
		c.pending.Delete(id)
		return true
	})

	c.mu.Lock()
	if c.conn != nil {
		// Set read deadline to immediately unblock any pending reads
		_ = c.conn.SetReadDeadline(time.Now())
		_ = c.conn.Close() // Ignore close errors - connection might already be closed
		c.conn = nil
	}
	if c.writeChan != nil {
		close(c.writeChan) // Signal writeLoop to exit
		c.writeChan = nil
	}
	c.mu.Unlock()

	close(c.doneCh)
	close(c.reconnectCh)
	c.wg.Wait()
	return nil
}

// Call calls the requested method, passing an optional set of arguments.
// If v is not nil, the result will be unmarshaled into it.
// Prefer to use the type-safe API clients for normal operations.
func (c *Client) Call(ctx context.Context, method string, params []any, v any) error {
	msgID := c.msgID.Add(1)
	if _, ok := ctx.Deadline(); !ok {
		// Context doesn't have a timeout, apply the default.
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.opts.DefaultWriteTimeout)
		defer cancel()
	}

	req := &Request{
		JSONRPC: "2.0",
		ID:      msgID,
		Method:  method,
		Params:  params,
	}
	resultCh := make(chan Response, 1)

	c.pending.Store(msgID, resultCh)
	defer func() {
		ch, ok := c.pending.LoadAndDelete(msgID)
		if ok {
			close(ch)
		}
	}()

	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.writeChan == nil || c.closed.Load() {
		return fmt.Errorf("not connected")
	}

	select {
	case c.writeChan <- req:
		// Request queued successfully.
	case <-ctx.Done():
		return ctx.Err()
	}

	select {
	case err := <-c.errCh:
		return err
	case result, ok := <-resultCh:
		if !ok {
			// Channel was closed, client is shutting down.
			return fmt.Errorf("client closed")
		}
		if result.Error != nil {
			return result.Error
		}
		if v != nil {
			return result.Unmarshal(v)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// CallJob calls a job method and waits for completion.
// If v is not nil, the result will be unmarshaled into it.
// Prefer to use the type-safe API clients for normal operations.
func (c *Client) CallJob(ctx context.Context, method string, params []any, v any) error {
	var jobID int
	if err := c.Call(ctx, method, params, &jobID); err != nil {
		return fmt.Errorf("call %s: %w", method, err)
	}

	job, err := c.Job.Wait(ctx, jobID)
	if err != nil {
		return fmt.Errorf("wait for job %d (%s): %w", jobID, method, err)
	}

	if v != nil && job.Result != nil {
		resultBytes, err := json.Marshal(job.Result)
		if err != nil {
			return fmt.Errorf("marshal job result: %w", err)
		}
		if err := json.Unmarshal(resultBytes, v); err != nil {
			return fmt.Errorf("unmarshal job result: %w", err)
		}
	}

	return nil
}

func (c *Client) reconnect() error {
	if err := c.connect(); err != nil {
		return err
	}
	return c.authenticate()
}

func (c *Client) connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	u, err := url.Parse(c.url)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	dialer := &websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}

	conn, _, err := dialer.Dial(u.String(), http.Header{})
	if err != nil {
		return fmt.Errorf("websocket dial: %s: %w", u.String(), err)
	}

	c.conn = conn
	c.writeChan = make(chan *Request, 256)
	c.closed.Store(false)
	return nil
}

func (c *Client) authenticate() error {
	// Skip authentication if no credentials provided
	if c.opts.APIKey == "" && c.opts.Username == "" && c.opts.Password == "" {
		return nil
	}

	var method string
	var params []any

	if c.opts.APIKey != "" {
		method = "auth.login_with_api_key"
		params = []any{c.opts.APIKey}
	} else {
		method = "auth.login"
		params = []any{c.opts.Username, c.opts.Password}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	var success bool
	if err := c.Call(ctx, method, params, &success); err != nil {
		return fmt.Errorf("call %s: %w", method, err)
	}
	if success {
		return nil
	}
	return fmt.Errorf("auth unsuccessful")
}

func (c *Client) connectionManager() {
	defer c.wg.Done()
	defer func() {
		if c.opts.Debug {
			fmt.Println("connectionManager exiting")
		}
	}()

	c.wg.Add(2)
	c.mu.RLock()
	conn := c.conn
	writeChan := c.writeChan
	c.mu.RUnlock()
	go c.readLoop(conn)
	go c.writeLoop(conn, writeChan)

	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 100 * time.Millisecond
	bo.MaxInterval = 10 * time.Second
	bo.MaxElapsedTime = 0

	for !c.closed.Load() {
		<-c.reconnectCh
		if c.closed.Load() {
			return
		}

		if c.opts.Debug {
			fmt.Println("attempting to reconnect...")
		}

		if err := c.reconnect(); err != nil {
			if !c.closed.Load() {
				delay := bo.NextBackOff()
				if c.opts.Debug {
					fmt.Printf("reconnection failed, retrying in %s: %v\n", delay.String(), err)
				}
				select {
				case <-time.After(delay):
					if !c.closed.Load() {
						select {
						case c.reconnectCh <- struct{}{}:
						default:
						}
					}
				case <-c.doneCh:
					return
				}
			}
			continue
		}
		bo.Reset()
		if c.opts.Debug {
			fmt.Println("reconnected successfully")
		}

		c.wg.Add(2)
		c.mu.RLock()
		conn := c.conn
		writeChan := c.writeChan
		c.mu.RUnlock()
		go c.readLoop(conn)
		go c.writeLoop(conn, writeChan)
	}
}

func (c *Client) readLoop(conn *websocket.Conn) {
	defer c.wg.Done()
	defer func() {
		if c.opts.Debug {
			fmt.Println("readLoop exiting")
		}
	}()

	if conn == nil {
		return
	}

	for !c.closed.Load() {
		var resp Response
		if err := conn.ReadJSON(&resp); err != nil {
			if c.closed.Load() {
				return
			}
			// Check for connection errors that should trigger reconnection.
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) ||
				websocket.IsUnexpectedCloseError(err) ||
				strings.Contains(err.Error(), "connection reset") ||
				strings.Contains(err.Error(), "broken pipe") ||
				strings.Contains(err.Error(), "use of closed network connection") {
				if c.opts.Debug {
					fmt.Printf("connection lost: %v\n", err)
				}
				select {
				case c.reconnectCh <- struct{}{}:
				case <-c.doneCh:
				default:
				}
				return
			}
			if c.opts.Debug {
				fmt.Printf("recv err: %v\n", err)
			}
			select {
			case c.errCh <- fmt.Errorf("read message: %w", err):
			default:
			}
			continue
		}
		if c.opts.Debug {
			fmt.Printf("recv: %s\n", tryMarshal(resp))
		}

		if resp.ID != nil {
			// Response to a pending request.
			if ch, exists := c.pending.Load(*resp.ID); exists {
				ch <- resp
			}
		} else if resp.Method != "" {
			// Server notification (no ID).
			if c.opts.NotificationHandler != nil {
				c.opts.NotificationHandler(resp.Method, resp.Params)
			} else if c.opts.Debug {
				fmt.Printf("unhandled notification: %s\n", resp.Method)
			}
		}
	}
}

func (c *Client) writeLoop(conn *websocket.Conn, messages <-chan *Request) {
	defer c.wg.Done()
	defer func() {
		if c.opts.Debug {
			fmt.Println("writeLoop exiting")
		}
	}()

	if conn == nil {
		return
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-messages:
			if !ok {
				return
			}
			if c.opts.Debug {
				fmt.Printf("send: %s\n", tryMarshal(msg))
			}
			if err := conn.WriteJSON(msg); err != nil {
				if c.opts.Debug {
					fmt.Printf("writeLoop error: %v\n", err)
				}
				return
			}
		case <-ticker.C:
			if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second)); err != nil {
				if c.opts.Debug {
					fmt.Printf("ping error: %v\n", err)
				}
				return
			}
		}
	}
}

func tryMarshal(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func Ptr[T any](v T) *T {
	return &v
}

func value[T any](v *T) T {
	var zero T
	if v == nil {
		return zero
	}
	return *v
}

// Request is a JSON-RPC 2.0 request sent to the server.
type Request struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params,omitempty"`
}

// Response is a JSON-RPC 2.0 response or notification from the server.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
	Method  string          `json:"method,omitempty"` // Notifications only.
	Params  json.RawMessage `json:"params,omitempty"` // Notifications only.
}

// Unmarshal decodes the response result into v.
func (r *Response) Unmarshal(v any) error {
	if err := json.Unmarshal(r.Result, v); err != nil {
		return fmt.Errorf("unmarshal result: %s: %w", string(r.Result), err)
	}
	return nil
}

// RPCError is a JSON-RPC 2.0 error object.
type RPCError struct {
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Data    *RPCErrorData `json:"data,omitempty"`
}

// RPCErrorData contains TrueNAS-specific error details nested within the JSON-RPC error.
type RPCErrorData struct {
	Error       int            `json:"error"`
	ErrName     string         `json:"errname"`
	Reason      string         `json:"reason"`
	Trace       *RPCErrorTrace `json:"trace,omitempty"`
	Extra       []any          `json:"extra,omitempty"`
	PyException string         `json:"py_exception,omitempty"`
}

// RPCErrorTrace contains Python traceback information from TrueNAS.
type RPCErrorTrace struct {
	Class     string `json:"class"`
	Frames    []any  `json:"frames"`
	Formatted string `json:"formatted"`
	Repr      string `json:"repr"`
}

func (e *RPCError) Error() string {
	var parts []string
	if e.Code != 0 {
		parts = append(parts, fmt.Sprintf("code: %d", e.Code))
	}
	if e.Message != "" {
		parts = append(parts, fmt.Sprintf("message: %s", e.Message))
	}
	if e.Data != nil {
		if e.Data.Reason != "" {
			parts = append(parts, fmt.Sprintf("reason: %s", e.Data.Reason))
		}
		if e.Data.ErrName != "" {
			parts = append(parts, fmt.Sprintf("errname: %s", e.Data.ErrName))
		}
	}
	if len(parts) == 0 {
		return "TrueNAS API error"
	}
	return fmt.Sprintf("TrueNAS API error (%s)", strings.Join(parts, ", "))
}
