package libkb

import (
	"fmt"
	"io"
	"os"
	"strings"

	triplesec "github.com/keybase/go-triplesec"
	"golang.org/x/crypto/openpgp"
)

type KeyringFile struct {
	filename         string
	Entities         openpgp.EntityList
	isPublic         bool
	indexID          map[string](*openpgp.Entity) // Map of 64-bit uppercase-hex KeyIds
	indexFingerprint map[PGPFingerprint](*openpgp.Entity)
	Contextified
}

type Keyrings struct {
	Contextified
}

func (k *Keyrings) MakeKeyrings(filenames []string, isPublic bool) []*KeyringFile {
	v := make([]*KeyringFile, len(filenames), len(filenames))
	for i, filename := range filenames {
		v[i] = &KeyringFile{filename, openpgp.EntityList{}, isPublic, nil, nil, Contextified{g: k.G()}}
	}
	return v
}

func NewKeyrings(g *GlobalContext) *Keyrings {
	ret := &Keyrings{
		Contextified: Contextified{g: g},
	}
	return ret
}

//===================================================================

func (g *GlobalContext) SKBFilenameForUser(un NormalizedUsername) string {
	tmp := g.Env.GetSecretKeyringTemplate()
	token := "%u"
	if strings.Index(tmp, token) < 0 {
		return tmp
	}

	return strings.Replace(tmp, token, un.String(), -1)
}

// Note:  you need to be logged in as 'un' for this function to
// work.  Otherwise, it just silently returns nil, nil.
func LoadSKBKeyring(un NormalizedUsername, g *GlobalContext) (*SKBKeyringFile, error) {
	if un.IsNil() {
		return nil, NoUsernameError{}
	}

	skbfile := NewSKBKeyringFile(g.SKBFilenameForUser(un))
	err := skbfile.LoadAndIndex()
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return skbfile, nil
}

func (k *KeyringFile) LoadAndIndex() error {
	var err error
	k.G().Log.Debug("+ LoadAndIndex on %s", k.filename)
	if err = k.Load(); err == nil {
		err = k.Index()
	}
	k.G().Log.Debug("- LoadAndIndex on %s -> %s", k.filename, ErrToOk(err))
	return err
}

func (k *KeyringFile) Index() error {
	k.G().Log.Debug("+ Index on %s", k.filename)
	k.indexID = make(map[string](*openpgp.Entity))
	k.indexFingerprint = make(map[PGPFingerprint](*openpgp.Entity))
	p := 0
	s := 0
	for _, entity := range k.Entities {
		if entity.PrimaryKey != nil {
			id := entity.PrimaryKey.KeyIdString()
			k.indexID[id] = entity
			fp := PGPFingerprint(entity.PrimaryKey.Fingerprint)
			k.indexFingerprint[fp] = entity
			p++
		}
		for _, subkey := range entity.Subkeys {
			if subkey.PublicKey != nil {
				id := subkey.PublicKey.KeyIdString()
				k.indexID[id] = entity
				fp := PGPFingerprint(subkey.PublicKey.Fingerprint)
				k.indexFingerprint[fp] = entity
				s++
			}
		}
	}
	k.G().Log.Debug("| Indexed %d primary and %d subkeys", p, s)
	k.G().Log.Debug("- Index on %s -> %s", k.filename, "OK")
	return nil
}

func (k *KeyringFile) Load() error {
	k.G().Log.Debug(fmt.Sprintf("+ Loading PGP Keyring %s", k.filename))
	file, err := os.Open(k.filename)
	if os.IsNotExist(err) {
		G.Log.Warning(fmt.Sprintf("No PGP Keyring found at %s", k.filename))
		err = nil
	} else if err != nil {
		G.Log.Errorf("Cannot open keyring %s: %s\n", k.filename, err)
		return err
	}
	if file != nil {
		k.Entities, err = openpgp.ReadKeyRing(file)
		if err != nil {
			G.Log.Errorf("Cannot parse keyring %s: %s\n", k.filename, err)
			return err
		}
	}
	k.G().Log.Debug(fmt.Sprintf("- Successfully loaded PGP Keyring"))
	return nil
}

func (k KeyringFile) WriteTo(w io.Writer) (int64, error) {
	for _, e := range k.Entities {
		if err := e.Serialize(w); err != nil {
			return 0, err
		}
	}
	return 0, nil
}

func (k KeyringFile) GetFilename() string { return k.filename }

func (k KeyringFile) Save() error {
	return SafeWriteToFile(k, 0)
}

type SecretKeyType int

const (
	// The current device signing key.
	DeviceSigningKeyType SecretKeyType = iota
	// The current device encryption key.
	DeviceEncryptionKeyType
	// A PGP key (including the synced PGP key, if there is one).
	PGPKeyType
)

func (t SecretKeyType) String() string {
	switch t {
	case DeviceSigningKeyType:
		return "DeviceSigningKeyType"
	case DeviceEncryptionKeyType:
		return "DeviceEncryptionKeyType"
	case PGPKeyType:
		return "PGPKeyType"
	default:
		return "<Unknown secret key type>"
	}
}

func (t SecretKeyType) nonDeviceKeyMatches(key GenericKey) bool {
	if IsPGP(key) && (t == PGPKeyType) {
		return true
	}

	return false
}

type SecretKeyArg struct {
	// Whose keys to use. Must be non-nil.
	Me *User

	// The allowed key types.
	KeyType SecretKeyType

	// For non-device keys, a string that the key has to match. If
	// empty, any valid key is allowed.
	KeyQuery   string
	ExactMatch bool // if set, full equality required
}

// GetSecretKeyLocked gets a secret key for the current user by first
// looking for keys synced from the server, and if that fails, tries
// those in the local Keyring that are also active for the user.
// In any case, the key will be locked.
func (k *Keyrings) GetSecretKeyLocked(lctx LoginContext, ska SecretKeyArg) (ret *SKB, which string, err error) {
	k.G().Log.Debug("+ GetSecretKeyLocked()")
	defer func() {
		k.G().Log.Debug("- GetSecretKeyLocked() -> %s", ErrToOk(err))
	}()

	k.G().Log.Debug("| LoadMe w/ Secrets on")

	if ska.Me == nil {
		if ska.Me, err = LoadMe(NewLoadUserArg(k.G())); err != nil {
			return
		}
	}

	if lctx != nil {
		ret, err = lctx.LockedLocalSecretKey(ska)
		if err != nil {
			return ret, which, err
		}
	} else {
		aerr := k.G().LoginState().Account(func(a *Account) {
			ret, err = a.LockedLocalSecretKey(ska)
		}, "LockedLocalSecretKey")
		if err != nil {
			return ret, which, err
		}
		if aerr != nil {
			return nil, which, aerr
		}
	}

	if ret != nil {
		k.G().Log.Debug("| Getting local secret key")
		return
	}

	var pub GenericKey

	if ska.KeyType != PGPKeyType {
		k.G().Log.Debug("| Skipped Synced PGP key (via options)")
		err = NoSecretKeyError{}
		return
	}

	if ret, err = ska.Me.SyncedSecretKey(lctx); err != nil {
		k.G().Log.Warning("Error fetching synced PGP secret key: %s", err)
		return
	}
	if ret == nil {
		err = NoSecretKeyError{}
		return
	}

	if pub, err = ret.GetPubKey(); err != nil {
		return
	}

	if !KeyMatchesQuery(pub, ska.KeyQuery, ska.ExactMatch) {
		k.G().Log.Debug("| Can't use Synced PGP key; doesn't match query %s", ska.KeyQuery)
		err = NoSecretKeyError{}
		return nil, "", err

	}

	which = "your Keybase.io passphrase"
	return
}

// TODO: Figure out whether and how to dep-inject the SecretStore.
func (k *Keyrings) GetSecretKeyWithPrompt(lctx LoginContext, ska SecretKeyArg, secretUI SecretUI, reason string) (key GenericKey, skb *SKB, err error) {
	k.G().Log.Debug("+ GetSecretKeyWithPrompt(%s)", reason)
	defer func() {
		k.G().Log.Debug("- GetSecretKeyWithPrompt() -> %s", ErrToOk(err))
	}()
	var which string
	if skb, which, err = k.GetSecretKeyLocked(lctx, ska); err != nil {
		skb = nil
		return
	}
	var secretStore SecretStore
	if ska.Me != nil {
		skb.SetUID(ska.Me.GetUID())
		secretStore = NewSecretStore(ska.Me.GetNormalizedName())
	}
	if key, err = skb.PromptAndUnlock(lctx, reason, which, secretStore, secretUI, nil, ska.Me); err != nil {
		key = nil
		skb = nil
		return
	}
	return
}

func (k *Keyrings) GetSecretKeyWithStoredSecret(lctx LoginContext, ska SecretKeyArg, me *User, secretRetriever SecretRetriever) (key GenericKey, err error) {
	k.G().Log.Debug("+ GetSecretKeyWithStoredSecret()")
	defer func() {
		k.G().Log.Debug("- GetSecretKeyWithStoredSecret() -> %s", ErrToOk(err))
	}()
	var skb *SKB
	skb, _, err = k.GetSecretKeyLocked(lctx, ska)
	if err != nil {
		return
	}
	skb.SetUID(me.GetUID())
	return skb.UnlockWithStoredSecret(secretRetriever)
}

func (k *Keyrings) GetSecretKeyWithPassphrase(lctx LoginContext, me *User, passphrase string, secretStorer SecretStorer) (key GenericKey, err error) {
	k.G().Log.Debug("+ GetSecretKeyWithPassphrase()")
	defer func() {
		k.G().Log.Debug("- GetSecretKeyWithPassphrase() -> %s", ErrToOk(err))
	}()
	ska := SecretKeyArg{
		Me:      me,
		KeyType: DeviceSigningKeyType,
	}
	var skb *SKB
	skb, _, err = k.GetSecretKeyLocked(lctx, ska)
	if err != nil {
		return
	}
	skb.SetUID(me.GetUID())
	var tsec *triplesec.Cipher
	var pps *PassphraseStream
	if lctx != nil {
		tsec = lctx.PassphraseStreamCache().Triplesec()
		pps = lctx.PassphraseStreamCache().PassphraseStream()
	} else {
		k.G().LoginState().PassphraseStreamCache(func(sc *PassphraseStreamCache) {
			tsec = sc.Triplesec()
			pps = sc.PassphraseStream()
		}, "StreamCache - tsec, pps")
	}
	return skb.UnlockSecretKey(lctx, passphrase, tsec, pps, secretStorer, nil)
}

type EmptyKeyRing struct{}

func (k EmptyKeyRing) KeysById(id uint64) []openpgp.Key {
	return []openpgp.Key{}
}
func (k EmptyKeyRing) KeysByIdUsage(id uint64, usage byte) []openpgp.Key {
	return []openpgp.Key{}
}
func (k EmptyKeyRing) DecryptionKeys() []openpgp.Key {
	return []openpgp.Key{}
}
