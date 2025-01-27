package collection

import (
	"context"
	"sort"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/require"
)

func TestLoadFromManifest(t *testing.T) {
	expIDs := []string{
		"sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a",
		"sha256:5410e32d6fdd0b638ae95c0c88326a6afe62105f2db1505ded397d2074dcbeb5",
		"sha256:0c7f453f9f3463d41110402f70e913ef7d850986a231276c0065ff958639b976",
		"sha256:2e30f6131ce2164ed5ef017845130727291417d60a1be6fad669bdc4473289cd",
		"sha256:5c29ebcf4a3e7ac6dca6dcea98b4fa98de57c4aca65fa0b49989fbeab1dfdf84",
		"sha256:684853a5c4538a93eaee331454bdf152be9d4a54fee9be3121594adac335a3ab",
	}
	root := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageManifest,
		Digest:    digest.Digest("sha256:5410e32d6fdd0b638ae95c0c88326a6afe62105f2db1505ded397d2074dcbeb5"),
	}

	collection := New("test")
	err := LoadFromManifest(context.Background(), collection, testFetcher, root)
	require.NoError(t, err)
	var ids []string
	for _, node := range collection.Nodes() {
		ids = append(ids, node.ID())
	}
	sortIDS(ids)
	sortIDS(expIDs)
	require.Equal(t, expIDs, ids)
}

func sortIDS(ids []string) {
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})
}

func testFetcher(ctx context.Context, desc ocispec.Descriptor) ([]byte, error) {
	s := `{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.manifest.v1+json",
  "config": {
    "mediaType": "application/vnd.uor.config.v1+json",
    "digest": "sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a",
    "size": 2
  },
  "layers": [
    {
      "mediaType": "application/json",
      "digest": "sha256:0c7f453f9f3463d41110402f70e913ef7d850986a231276c0065ff958639b976",
      "size": 76,
      "annotations": {
        "org.opencontainers.image.title": "info.json"
      }
    },
    {
      "mediaType": "image/jpeg",
      "digest": "sha256:2e30f6131ce2164ed5ef017845130727291417d60a1be6fad669bdc4473289cd",
      "size": 5536,
      "annotations": {
        "org.opencontainers.image.title": "images/fish.jpg"
      }
    },
    {
      "mediaType": "application/json",
      "digest": "sha256:5c29ebcf4a3e7ac6dca6dcea98b4fa98de57c4aca65fa0b49989fbeab1dfdf84",
      "size": 32,
      "annotations": {
        "org.opencontainers.image.title": "supplementary/about.json"
      }
    },
    {
      "mediaType": "application/json",
      "digest": "sha256:684853a5c4538a93eaee331454bdf152be9d4a54fee9be3121594adac335a3ab",
      "size": 59,
      "annotations": {
        "org.opencontainers.image.title": "test.json"
      }
    }
  ],
  "annotations": {
    "uor.schema": "localhost:5001/schema:latest"
  }
}
`
	if desc.Digest.String() == "sha256:5410e32d6fdd0b638ae95c0c88326a6afe62105f2db1505ded397d2074dcbeb5" {
		return []byte(s), nil
	}
	return []byte{}, nil
}
