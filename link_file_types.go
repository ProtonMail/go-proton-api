package proton

import (
	"encoding/json"

	"github.com/ProtonMail/gopenpgp/v2/crypto"
)

type CreateFileReq struct {
	ParentLinkID string

	Name     string // Encrypted File Name
	Hash     string // Encrypted content hash
	MIMEType string // MIME Type

	ContentKeyPacket          string // The block's key packet, encrypted with the node key.
	ContentKeyPacketSignature string // Unencrypted signature of the content session key, signed with the NodeKey

	NodeKey                 string // The private NodeKey, used to decrypt any file/folder content.
	NodePassphrase          string // The passphrase used to unlock the NodeKey, encrypted by the owning Link/Share keyring.
	NodePassphraseSignature string // The signature of the NodePassphrase

	SignatureAddress string // Signature email address used to sign passphrase and name
}

type CreateFileRes struct {
	ID         string // Encrypted Link ID
	RevisionID string // Encrypted Revision ID
}

type UpdateRevisionReq struct {
	BlockList         []BlockToken
	State             RevisionState
	ManifestSignature string
	SignatureAddress  string
	XAttr             string
}

type RevisionXAttrCommon struct {
	ModificationTime string
	Size             int64
	BlockSizes       []int64
	Digests          map[string]string
	// Mode removed — migrated to the POSIX section (proton-utils' PosixXAttr).
}

// RevisionXAttr is the decoded XAttr blob. Common is Proton's typed schema;
// Extra preserves every other top-level section verbatim so a decode→encode
// round-trip never drops data written by other clients (Media, Camera,
// Location) or the shared POSIX section. Callers read/write their own
// namespaced sections through Extra.
type RevisionXAttr struct {
	Common RevisionXAttrCommon
	Extra  map[string]json.RawMessage `json:"-"` // unmodeled top-level sections
}

// revisionXAttrAlias has the same modeled fields but no Marshal/Unmarshal
// methods, so encoding/json can (de)serialize Common without recursing into
// RevisionXAttr's custom codec.
type revisionXAttrAlias struct {
	Common RevisionXAttrCommon
}

// MarshalJSON encodes the typed Common section plus every section held in
// Extra. Top-level keys are emitted in ascending lexicographic order (map-key
// sorting by encoding/json), so output is byte-deterministic for a given set
// of sections. The typed Common wins over any stray Extra["Common"].
func (x RevisionXAttr) MarshalJSON() ([]byte, error) {
	m := make(map[string]json.RawMessage, len(x.Extra)+1)
	for k, v := range x.Extra {
		m[k] = v
	}

	common, err := json.Marshal(revisionXAttrAlias{Common: x.Common})
	if err != nil {
		return nil, err
	}

	// common is `{"Common":{...}}`; lift its single member into m so the typed
	// Common overrides any Extra["Common"].
	var tmp map[string]json.RawMessage
	if err := json.Unmarshal(common, &tmp); err != nil {
		return nil, err
	}
	for k, v := range tmp {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON decodes the Common section into the typed field and places
// every other top-level section into Extra as verbatim raw JSON. A blob with
// no Common yields a zero Common; an empty object leaves Extra nil.
func (x *RevisionXAttr) UnmarshalJSON(data []byte) error {
	var a revisionXAttrAlias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	x.Common = a.Common

	var all map[string]json.RawMessage
	if err := json.Unmarshal(data, &all); err != nil {
		return err
	}
	delete(all, "Common")
	if len(all) > 0 {
		x.Extra = all
	}

	return nil
}

// SetEncXAttrString marshals the full RevisionXAttr (Common + Extra),
// encrypts it with the node keyring, signs it with the address keyring, and
// stores the armored result on the request. The custom MarshalJSON emits every
// unmodeled section held in Extra (e.g. the POSIX section), so callers write
// their namespaced metadata through Extra.
func (updateRevisionReq *UpdateRevisionReq) SetEncXAttrString(addrKR, nodeKR *crypto.KeyRing, xAttr *RevisionXAttr) error {
	// Source
	// - https://github.com/ProtonMail/WebClients/blob/099a2451b51dea38b5f0e07ec3b8fcce07a88303/packages/shared/lib/interfaces/drive/link.ts#L53
	// - https://github.com/ProtonMail/WebClients/blob/main/applications/drive/src/app/store/_links/extendedAttributes.ts#L139
	// XAttr has following JSON structure encrypted by node key:
	// {
	//    Common: {
	//        ModificationTime: "2021-09-16T07:40:54+0000",
	//        Size: 13283,
	// 		  BlockSizes: [1,2,3],
	//        Digests: "sha1 string"
	//    },
	// }

	jsonByteArr, err := json.Marshal(xAttr)
	if err != nil {
		return err
	}

	encXattr, err := nodeKR.Encrypt(crypto.NewPlainMessage(jsonByteArr), addrKR)
	if err != nil {
		return err
	}

	encXattrString, err := encXattr.GetArmored()
	if err != nil {
		return err
	}

	updateRevisionReq.XAttr = encXattrString
	return nil
}

type BlockToken struct {
	Index int
	Token string
}
