package executor

import (
	"context"
	"io"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type imageCache struct {
	pulled sync.Map
}

// ensure hace pull de la imagen si no está en caché local.
// Usa sync.Map para evitar pulls concurrentes de la misma imagen.
func (c *imageCache) ensure(ctx context.Context, cli *client.Client, imageName string) error {
	if _, ok := c.pulled.Load(imageName); ok {
		return nil
	}

	rc, err := cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer rc.Close()
	io.Copy(io.Discard, rc) // drena el stream para que complete

	c.pulled.Store(imageName, struct{}{})
	return nil
}
