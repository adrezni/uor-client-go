package cli

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/credentials"
	"github.com/docker/cli/cli/config/types"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/sigstore/cosign/cmd/cosign/cli/options"
	"github.com/sigstore/cosign/cmd/cosign/cli/sign"
	"github.com/sigstore/cosign/cmd/cosign/cli/verify"

	// Loads OIDC providers
	_ "github.com/sigstore/cosign/pkg/providers/all"
)

// Sign applies keyless OIDC signatures to sign UOR Collections
func signCollection(ctx context.Context, o *PushOptions) error {

	ko := options.KeyOpts{
		RekorURL:        "https://rekor.sigstore.dev",
		OIDCClientID:    "sigstore",
		OIDCRedirectURL: "",
		OIDCIssuer:      "https://oauth2.sigstore.dev/auth",
		FulcioURL:       "https://fulcio.sigstore.dev",
	}

	// Required by sigstore / cosign for keyless signing at the time of writing
	os.Setenv("COSIGN_EXPERIMENTAL", "1")

	regopts := options.RegistryOptions{
		Keychain: authn.DefaultKeychain,
	}
	if o.PlainHTTP || o.Insecure {
		regopts.AllowInsecure = true
	}

	if len(o.Configs) != 0 {
		var err error
		regopts.Keychain, err = buildKeychain(o.Configs)
		if err != nil {
			return err
		}
	}

	// Note(afflom): Setting this bool doesn't do anything. Regardless of
	// the boolean's value, the output is always debug. Waiting for
	// https://github.com/sigstore/cosign/issues/844.
	var llevel bool
	if o.LogLevel == "debug" {
		llevel = true
	}

	opts := options.RootOptions{
		Verbose: llevel,
		Timeout: 100 * time.Second,
	}
	err := sign.SignCmd(&opts, ko, regopts, map[string]interface{}{},
		[]string{o.Destination}, "", "", true, "", "",
		"", true, false, "", false)
	if err != nil {
		return fmt.Errorf("getting signer: %w", err)
	}
	return nil

}

// Verify performs signature verification of keyless signatures
func verifyCollection(o *PullOptions, ctx context.Context) error {

	regopts := options.RegistryOptions{
		Keychain: authn.DefaultKeychain,
	}

	if o.PlainHTTP || o.Insecure {
		regopts.AllowInsecure = true
	}

	if len(o.Configs) != 0 {
		var err error
		regopts.Keychain, err = buildKeychain(o.Configs)
		if err != nil {
			return err
		}
	}

	v := verify.VerifyCommand{
		RekorURL:        "https://rekor.sigstore.dev",
		RegistryOptions: regopts,
	}

	// Required by sigstore / cosign for keyless signing at the time of writing
	os.Setenv("COSIGN_EXPERIMENTAL", "1")

	if err := v.Exec(ctx, []string{o.Source}); err != nil {
		return err
	}
	return nil
}

type KeyChainFunc func(authn.Resource) (authn.Authenticator, error)

func (fn KeyChainFunc) Resolve(r authn.Resource) (authn.Authenticator, error) {
	return fn(r)
}

func buildKeychain(c []string) (authn.Keychain, error) {
	var keychainFuncs []authn.Keychain
	var mu sync.Mutex
	for _, config := range c {
		fromConfig := KeyChainFunc(func(target authn.Resource) (authn.Authenticator, error) {
			mu.Lock()
			defer mu.Unlock()
			cf := configfile.New(config)
			if _, err := os.Stat(config); err != nil {
				if !os.IsNotExist(err) {
					return nil, err
				}
			}

			file, err := os.Open(config)
			if err != nil {
				return nil, err
			}
			defer file.Close()
			if err := cf.LoadFromReader(file); err != nil {
				return nil, err
			}

			if !cf.ContainsAuth() {
				cf.CredentialsStore = credentials.DetectDefaultStore(cf.CredentialsStore)
			}

			// See:
			// https://github.com/google/ko/issues/90
			// https://github.com/moby/moby/blob/fc01c2b481097a6057bec3cd1ab2d7b4488c50c4/registry/config.go#L397-L404
			var cfg, empty types.AuthConfig
			for _, key := range []string{
				target.String(),
				target.RegistryStr(),
			} {
				if key == name.DefaultRegistry {
					key = authn.DefaultAuthKey
				}

				cfg, err = cf.GetAuthConfig(key)
				if err != nil {
					return nil, err
				}
				if cfg != empty {
					break
				}
			}
			if cfg == empty {
				return authn.Anonymous, nil
			}

			return authn.FromConfig(authn.AuthConfig{
				Username:      cfg.Username,
				Password:      cfg.Password,
				Auth:          cfg.Auth,
				IdentityToken: cfg.IdentityToken,
				RegistryToken: cfg.RegistryToken,
			}), nil

		})
		keychainFuncs = append(keychainFuncs, fromConfig)

	}
	return authn.NewMultiKeychain(keychainFuncs...), nil
}
