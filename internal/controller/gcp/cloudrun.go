package controller

import (
	"context"

	gcprun "cloud.google.com/go/run/apiv2"
	"google.golang.org/api/option"
)

type newCloudRunServiceClient func(ctx context.Context, opts ...option.ClientOption) (*gcprun.ServicesClient, error)
