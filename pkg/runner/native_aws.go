package runner

import (
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/backup"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// NativeAWSRunner implements AWSRunner using aws-sdk-go-v2 directly.
type NativeAWSRunner struct {
	cfg aws.Config
	mu  sync.RWMutex

	// per-region client caches
	efsClients    map[string]*efs.Client
	backupClients map[string]*backup.Client
	ec2Clients    map[string]*ec2.Client
	elbClients    map[string]*elasticloadbalancing.Client
	elbv2Clients  map[string]*elasticloadbalancingv2.Client
	eksClients    map[string]*eks.Client
	rdsClients    map[string]*rds.Client
	logsClients   map[string]*cloudwatchlogs.Client

	// global clients (no region needed)
	stsClient *sts.Client
	iamClient *iam.Client
	s3Client  *s3.Client
}

func newNativeAWSRunner(ctx context.Context) (*NativeAWSRunner, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("native aws: load config: %w", err)
	}
	return &NativeAWSRunner{
		cfg:           cfg,
		efsClients:    make(map[string]*efs.Client),
		backupClients: make(map[string]*backup.Client),
		ec2Clients:    make(map[string]*ec2.Client),
		elbClients:    make(map[string]*elasticloadbalancing.Client),
		elbv2Clients:  make(map[string]*elasticloadbalancingv2.Client),
		eksClients:    make(map[string]*eks.Client),
		rdsClients:    make(map[string]*rds.Client),
		logsClients:   make(map[string]*cloudwatchlogs.Client),
		stsClient:     sts.NewFromConfig(cfg),
		iamClient:     iam.NewFromConfig(cfg),
		s3Client:      s3.NewFromConfig(cfg),
	}, nil
}

// --- Client accessors (double-checked locking) ---

func (r *NativeAWSRunner) efsClient(region string) *efs.Client {
	r.mu.RLock()
	if c, ok := r.efsClients[region]; ok {
		r.mu.RUnlock()
		return c
	}
	r.mu.RUnlock()
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.efsClients[region]; ok {
		return c
	}
	c := efs.NewFromConfig(r.cfg, func(o *efs.Options) { o.Region = region })
	r.efsClients[region] = c
	return c
}

func (r *NativeAWSRunner) backupClient(region string) *backup.Client {
	r.mu.RLock()
	if c, ok := r.backupClients[region]; ok {
		r.mu.RUnlock()
		return c
	}
	r.mu.RUnlock()
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.backupClients[region]; ok {
		return c
	}
	c := backup.NewFromConfig(r.cfg, func(o *backup.Options) { o.Region = region })
	r.backupClients[region] = c
	return c
}

func (r *NativeAWSRunner) ec2Client(region string) *ec2.Client {
	r.mu.RLock()
	if c, ok := r.ec2Clients[region]; ok {
		r.mu.RUnlock()
		return c
	}
	r.mu.RUnlock()
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.ec2Clients[region]; ok {
		return c
	}
	c := ec2.NewFromConfig(r.cfg, func(o *ec2.Options) { o.Region = region })
	r.ec2Clients[region] = c
	return c
}

func (r *NativeAWSRunner) elbClient(region string) *elasticloadbalancing.Client {
	r.mu.RLock()
	if c, ok := r.elbClients[region]; ok {
		r.mu.RUnlock()
		return c
	}
	r.mu.RUnlock()
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.elbClients[region]; ok {
		return c
	}
	c := elasticloadbalancing.NewFromConfig(r.cfg, func(o *elasticloadbalancing.Options) { o.Region = region })
	r.elbClients[region] = c
	return c
}

func (r *NativeAWSRunner) elbv2Client(region string) *elasticloadbalancingv2.Client {
	r.mu.RLock()
	if c, ok := r.elbv2Clients[region]; ok {
		r.mu.RUnlock()
		return c
	}
	r.mu.RUnlock()
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.elbv2Clients[region]; ok {
		return c
	}
	c := elasticloadbalancingv2.NewFromConfig(r.cfg, func(o *elasticloadbalancingv2.Options) { o.Region = region })
	r.elbv2Clients[region] = c
	return c
}

func (r *NativeAWSRunner) eksClient(region string) *eks.Client {
	r.mu.RLock()
	if c, ok := r.eksClients[region]; ok {
		r.mu.RUnlock()
		return c
	}
	r.mu.RUnlock()
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.eksClients[region]; ok {
		return c
	}
	c := eks.NewFromConfig(r.cfg, func(o *eks.Options) { o.Region = region })
	r.eksClients[region] = c
	return c
}

func (r *NativeAWSRunner) rdsClient(region string) *rds.Client {
	r.mu.RLock()
	if c, ok := r.rdsClients[region]; ok {
		r.mu.RUnlock()
		return c
	}
	r.mu.RUnlock()
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.rdsClients[region]; ok {
		return c
	}
	c := rds.NewFromConfig(r.cfg, func(o *rds.Options) { o.Region = region })
	r.rdsClients[region] = c
	return c
}

func (r *NativeAWSRunner) logsClient(region string) *cloudwatchlogs.Client {
	r.mu.RLock()
	if c, ok := r.logsClients[region]; ok {
		r.mu.RUnlock()
		return c
	}
	r.mu.RUnlock()
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.logsClients[region]; ok {
		return c
	}
	c := cloudwatchlogs.NewFromConfig(r.cfg, func(o *cloudwatchlogs.Options) { o.Region = region })
	r.logsClients[region] = c
	return c
}

// CheckVersion always returns nil for native mode (no external binary needed).
func (r *NativeAWSRunner) CheckVersion(ctx context.Context) error {
	return nil
}
