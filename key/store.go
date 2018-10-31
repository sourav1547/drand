package key

import (
	"errors"
	"os"
	"path"
	"reflect"

	"github.com/BurntSushi/toml"
	"github.com/dedis/drand/fs"
	"github.com/nikkolasg/slog"
)

// Store abstracts the loading and saving of any private/public cryptographic
// material to be used by drand. For the moment, only a file based store is
// implemented.
type Store interface {
	// SaveKeyPair saves the private key generated by drand as well as the
	// public identity key associated
	SaveKeyPair(p *Pair) error
	// LoadKeyPair loads the private/public key pair associated with the drand
	// operator
	LoadKeyPair() (*Pair, error)
	SaveShare(share *Share) error
	LoadShare() (*Share, error)
	SaveGroup(*Group) error
	LoadGroup() (*Group, error)
	SaveDistPublic(d *DistPublic) error
	LoadDistPublic() (*DistPublic, error)
}

// ErrStoreFile returns an error in case the store can not save the requested
// file
var ErrStoreFile = errors.New("store file issues")

// ErrAbsent returns
var ErrAbsent = errors.New("store can't find requested object")

// ConfigFolderFlag holds the name of the flag to set using the CLI to change
// the default configuration folder of drand. It mimicks the gpg flag option.
const ConfigFolderFlag = "homedir"

const KeyFolderName = "key"
const GroupFolderName = "groups"
const keyFileName = "drand_id"
const privateExtension = ".private"
const publicExtension = ".public"
const groupFileName = "drand_group.toml"
const shareFileName = "dist_key.private"
const distKeyFileName = "dist_key.public"

// Tomler represents any struct that can be (un)marshalled into/from toml format
type Tomler interface {
	TOML() interface{}
	FromTOML(i interface{}) error
	TOMLValue() interface{}
}

// fileStore is a Store using filesystem to store informations
type fileStore struct {
	baseFolder     string
	privateKeyFile string
	publicKeyFile  string
	shareFile      string
	distKeyFile    string
	groupFile      string
}

// NewDefaultFileStore is used to create the config folder and all the subfolders.
// If a folder alredy exists, we simply check the rights
func NewFileStore(baseFolder string) Store {
	//config folder
	if fs.CreateSecureFolder(baseFolder) == "" {
		slog.Fatal("Something went wrong with the config folder. Make sure that you have the appropriate rights.")
	}
	store := &fileStore{baseFolder: baseFolder}
	keyFolder := fs.CreateSecureFolder(path.Join(baseFolder, KeyFolderName))
	groupFolder := fs.CreateSecureFolder(path.Join(baseFolder, GroupFolderName))
	store.privateKeyFile = path.Join(keyFolder, keyFileName) + privateExtension
	store.publicKeyFile = path.Join(keyFolder, keyFileName) + publicExtension
	store.groupFile = path.Join(groupFolder, groupFileName)
	store.shareFile = path.Join(groupFolder, shareFileName)
	store.distKeyFile = path.Join(groupFolder, distKeyFileName)
	return store
}

// SaveKeyPair first saves the private key in a file with tight permissions and then
// saves the public part in another file.
func (f *fileStore) SaveKeyPair(p *Pair) error {
	if err := Save(f.privateKeyFile, p, true); err != nil {
		return err
	}
	slog.Infof("Saved the key : %s at %s", p.Public.Addr, f.publicKeyFile)
	return Save(f.publicKeyFile, p.Public, false)
}

// LoadKeyPair decode private key first then public
func (f *fileStore) LoadKeyPair() (*Pair, error) {
	p := new(Pair)
	if err := Load(f.privateKeyFile, p); err != nil {
		return nil, err
	}
	return p, Load(f.publicKeyFile, p.Public)
}

func (f *fileStore) LoadGroup() (*Group, error) {
	g := new(Group)
	return g, Load(f.groupFile, g)
}

func (f *fileStore) SaveGroup(g *Group) error {
	return Save(f.groupFile, g, false)
}

func (f *fileStore) SaveShare(share *Share) error {
	slog.Info("crypto store: saving private share in ", f.shareFile)
	return Save(f.shareFile, share, true)
}

func (f *fileStore) LoadShare() (*Share, error) {
	s := new(Share)
	return s, Load(f.shareFile, s)
}

func (f *fileStore) SaveDistPublic(d *DistPublic) error {
	slog.Info("crypto store: saving public distributed key in ", f.distKeyFile)
	return Save(f.distKeyFile, d, false)
}

func (f *fileStore) LoadDistPublic() (*DistPublic, error) {
	d := new(DistPublic)
	return d, Load(f.distKeyFile, d)
}

func Save(path string, t Tomler, secure bool) error {
	var fd *os.File
	var err error
	if secure {
		fd, err = fs.CreateSecureFile(path)
	} else {
		fd, err = os.Create(path)
	}
	if err != nil {
		slog.Infof("config: can't save %s to %s: %s", reflect.TypeOf(t).String(), path, err)
		return err
	}
	defer fd.Close()
	return toml.NewEncoder(fd).Encode(t.TOML())
}

func Load(path string, t Tomler) error {
	tomlValue := t.TOMLValue()
	var err error
	if _, err = toml.DecodeFile(path, tomlValue); err != nil {
		return err
	}
	return t.FromTOML(tomlValue)
}
