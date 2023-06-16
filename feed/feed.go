package feed

import (
	"bytes"
	"context"
	"encoding/json"
	"os"

	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
	ipldlegacy "github.com/ipfs/go-ipld-legacy"
	logging "github.com/ipfs/go-log/v2"
	"github.com/ipld/go-ipld-prime/codec/dagjson"
	"github.com/ipld/go-ipld-prime/node/bindnode"
	mh "github.com/multiformats/go-multihash"

	"github.com/allisterb/patr/blockchain"
	"github.com/allisterb/patr/did"
	"github.com/allisterb/patr/ipfs"
	"github.com/allisterb/patr/node"
)

type User struct {
	Did          string
	NostrPrivKey []byte
	NostrPubkey  []byte
	IPNSPrivKey  []byte
	IPNSPubKey   []byte
}

type Profile struct {
	Did string
}

var log = logging.Logger("patr/feed")

func init() {
	cbornode.RegisterCborType(Profile{})
}

func LoadUser(file string) (User, error) {
	_, err := os.Stat(file)
	if err != nil {
		log.Errorf("user file %v does not exist", err)
		return User{}, err
	}
	u, err := os.ReadFile(file)
	if err != nil {
		log.Errorf("could not read user file: %v", err)
		return User{}, err
	}
	var user User
	if json.Unmarshal(u, &user) != nil {
		log.Errorf("could not read JSON data from user file: %v", err)
		return User{}, err
	}
	return user, nil
}
func CreateProfile(ctx context.Context) error {
	d, err := did.Parse(node.CurrentConfig.Did)
	if err != nil {
		log.Errorf("could not parse DID %s: %v", node.CurrentConfig.Did, err)
		return err
	}
	log.Infof("creating patr profile for %s...", node.CurrentConfig.Did)
	node.PanicIfNotInitialized()
	_, err = blockchain.ResolveENS(d.ID.ID, node.CurrentConfig.InfuraSecretKey)
	if err != nil {
		log.Errorf("could not resolve ENS name %s", node.CurrentConfig.Did)
		return err
	}
	ipfsNode, ipfsShutdown, err := ipfs.StartIPFSNode(ctx, node.CurrentConfig.IPFSPrivKey, node.CurrentConfig.IPFSPubKey)
	if err != nil {
		return err
	}
	profile := Profile{Did: node.CurrentConfig.Did}
	dagnode := bindnode.Wrap(&profile, nil)
	var buf bytes.Buffer
	err = dagjson.Encode(dagnode, &buf)
	if err != nil {
		log.Errorf("error encoding DAG node for profile %v as DAG-JSON: %v", profile.Did, err)
		ipfsShutdown()
		return err
	}
	cidprefix := cid.Prefix{
		Version:  1, // Usually '1'.
		Codec:    cid.DagJSON,
		MhType:   mh.SHA3_384, // 0x15 means "sha3-384" -- See the multicodecs table: https://github.com/multiformats/multicodec/
		MhLength: 48,          // sha3-384 hash has a 48-byte sum.
	}
	xcid, err := cidprefix.Sum(buf.Bytes())
	if err != nil {
		log.Errorf("error creating CID for DAG node for profile %v as DAG-JSON: %v", profile.Did, err)
		ipfsShutdown()
		return err
	}
	blk, err := blocks.NewBlockWithCid(buf.Bytes(), xcid)
	if err != nil {
		log.Errorf("error creating IPFS block for DAG node for profile %v as DAG-JSON: %v", profile.Did, err)
		ipfsShutdown()
		return err
	}
	log.Infof("IPFS block cid for DAG node for profile %s : %s", profile.Did, blk.Cid())
	err = ipfsNode.Dag().Pinning().Add(ctx, &ipldlegacy.LegacyNode{blk, dagnode})
	if err != nil {

		log.Errorf("error pinning IPFS block %v for DAG node for profile %v: %v", blk.Cid(), profile.Did, err)
		ipfsShutdown()
		return err
	}
	ipfs.PublishIPNSRecordForDAGNode(ctx, ipfsNode, blk.Cid())
	_, err = ipfs.PutIPFSDAGBlockToW3S(ctx, ipfsNode, node.CurrentConfig.W3SSecretKey, blk)
	if err != nil {
		log.Errorf("could not pin IPFS block %v using Web3.Storage service")
		ipfsShutdown()
		return err
	}
	err = ipfs.PublishIPNSRecordForDAGNodeToW3S(ctx, node.CurrentConfig.W3SSecretKey, blk.Cid(), node.CurrentConfig.IPFSPrivKey, node.CurrentConfig.IPFSPubKey)
	ipfsShutdown()
	return err
}
