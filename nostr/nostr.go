package nostr

import (
	"fmt"
	"time"

	"net/http"

	"github.com/allisterb/patr/ipfs"
	"github.com/fiatjaf/relayer"
	logging "github.com/ipfs/go-log/v2"
	"github.com/nbd-wtf/go-nostr"

	iface "github.com/ipfs/boxo/coreiface"
)

var log = logging.Logger("patr/nostr")

func GenerateKeyPair() (string, string, error) {
	sk := nostr.GeneratePrivateKey()
	pk, err := nostr.GetPublicKey(sk)
	if err != nil {
		log.Errorf("Error generating secp256k1 keypair: %v", err)
		return "", "", err
	}
	return sk, pk, err
}

type Logger struct {
}

type Relay struct {
	Ipfs ipfs.IPFSCore
}

type Storage struct {
	ipfs iface.CoreAPI
}

func (l *Logger) Infof(format string, v ...any) {
	log.Infof(format, v)
}

func (l *Logger) Warningf(format string, v ...any) {
	log.Warnf(format, v)
}

func (l *Logger) Errorf(format string, v ...any) {
	log.Errorf(format, v)
}

func (s *Storage) SaveEvent(evt *nostr.Event) error {
	//evt.Kind == nostr.
	return nil
}
func (r *Relay) Name() string {
	return "PatrRelay"
}

func (r *Relay) Init() error {
	log.Infof("patr relay initializing...")
	return nil
}

func (r *Relay) OnInitialized(s *relayer.Server) {
	// special handlers
	//s.Router().Path("/").HandlerFunc(handleWebpage)
	s.Router().Path("/dm").HandlerFunc(func(w http.ResponseWriter, rq *http.Request) {

	})
	log.Info("patr relay initialized")
}

func CreateTestEvent(privkey string, text string, ipfscore ipfs.IPFSCore) error {
	e := nostr.Event{
		ID:        "0",
		Content:   text,
		CreatedAt: nostr.Timestamp(time.Now().UnixMicro()),
		Kind:      nostr.KindApplicationSpecificData,
	}
	err := e.Sign(privkey)
	if err != nil {
		log.Errorf("could not sign test event: %v", err)
		return err
	}
	sc, err := e.CheckSignature()
	if (!sc) || err != nil {
		return fmt.Errorf("signing test event failed")
	}

	l, err := ipfs.PutNostrEventAsIPLDLink(ipfscore.Ctx, ipfscore, e)
	if err != nil {
		return fmt.Errorf("could not create test event %v with text %s: %v", e.ID, e.Content, err)
	} else {
		log.Infof("created event %v with text %s at https://ipfs.io/ipfs/%v", e.ID, e.Content, l.String())
		return nil
	}
}
