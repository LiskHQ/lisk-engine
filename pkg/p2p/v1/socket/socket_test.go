package socket

import (
	"testing"
)

func TestClientSocketListen(t *testing.T) {
	// connector := func() (*websocket.Conn, error) {
	// 	u := url.URL{
	// 		Scheme:   "ws",
	// 		Host:     "127.0.0.1:5001",
	// 		Path:     "/rpc/",
	// 		RawQuery: fmt.Sprintf("chainID=%s&networkVersion=%s&nonce=%s&port=%d", "01e47ba4e3e57981642150f4b45f64c2160c10bac9434339888210a4fa5df097", "2.0", "O2wTkjqplHIIabab", 4000),
	// 	}
	// 	fmt.Println(u.String())
	// 	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	// 	return conn, err
	// }
	// conn, err := connector()
	// assert.Nil(t, err)
	// ctx := context.Background()
	// sock, err := NewClientSocket(ctx, conn, connector, log.DefaultLogger)
	// assert.Nil(t, err)
	// ch := make(chan bool)
	// sub := sock.Subscribe()
	// go func() {
	// 	for msg := range sub {
	// 		switch msg.Kind() {
	// 		case EventSocketError:
	// 			t.Error(msg.Err())
	// 		case EventMessageReceived:
	// 			t.Log("received message", msg.Event(), msg.Data())
	// 		default:
	// 			t.Log("others", msg.Kind())
	// 		}
	// 	}
	// 	t.Log("Closed successfully")
	// 	close(ch)
	// }()
	// <-time.After(10 * time.Second)
	// sock.Close()
	// <-ch

	// goleak.VerifyNone(t)
	// t.FailNow()
}
