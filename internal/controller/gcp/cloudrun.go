package controller

import (
	gcprun "cloud.google.com/go/run/apiv2"
	"context"
	"google.golang.org/api/option"
)

type newCloudRunServiceClient func(ctx context.Context, opts ...option.ClientOption) (*gcprun.ServicesClient, error)
