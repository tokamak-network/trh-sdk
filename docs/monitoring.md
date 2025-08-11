# Monitoring Plugin

The Monitoring plugin provides comprehensive monitoring, alerting and loggic capabilities for the Thanos Stack.

### Tech Stack
- **Prometheus**: Metrics collection and storage with custom recording rules
- **Grafana**: Visualization dashboards with custom panels and CloudWatch integration
- **AlertManager**: Alert routing and notification with email/telegram channels
- **Blackbox Exporter**: External endpoint monitoring and health checks
- **AWS CLI Sidecar**: Log collection and forwarding to CloudWatch Logs
- **CloudWatch Logs**: Centralized log storage and retention management

Please check the user guide here.

## Table of Contents

- [Installation](#installation)
- [Uninstallation](#uninstallation)
- [Sub-Command Set: Alert Customization](#alert-customization)
- [Sub-Command Set: Log Collection](#log-collection)


## Installation

To install the monitoring plugin:

```bash
# Install monitoring plugin
# During installation, you can choose whether to use email and Telegram notification channels. You can also choose whether to use log collection.
trh-sdk install monitoring
```

## Uninstallation

To uninstall the monitoring plugin:

```bash
# Uninstall monitoring plugin
trh-sdk uninstall monitoring
```


## Alert Customization

### Quick Start
```bash
# Check current alert status
trh-sdk alert-config --status

# Configure email alerts
trh-sdk alert-config --channel email --configure

# Configure telegram alerts  
trh-sdk alert-config --channel telegram --configure

# Configure alert rules interactively
trh-sdk alert-config --rule set

# Reset all rules to default values
trh-sdk alert-config --rule reset
```

### Features
- **Email & Telegram Notifications**: Set up multiple notification channels
- **Configurable Alert Rules**: Adjust thresholds for balance, CPU, memory, and more
- **Interactive Configuration**: User-friendly command-line interface
- **Status Monitoring**: Real-time alert status and configuration details

## Log Collection

### Quick Start
```bash
# Enable CloudWatch log collection
trh-sdk log-collection --enable

# Configure log retention period (default: 30 days)
trh-sdk log-collection --retention 30

# Configure collection interval (default: 30 seconds)
trh-sdk log-collection --interval 60

# Show current logging configuration
trh-sdk log-collection --show

# Disable logging
trh-sdk log-collection --disable

# Download logs from running components
trh-sdk log-collection --download --component op-node --hours 1 --keyword error

# Download logs for all components
trh-sdk log-collection --download --component all --hours 24 --keyword warning
```

### Features
- **CloudWatch Storage**: Logs are automatically stored in CloudWatch Log Groups with the format `/aws/eks/{namespace}/{component}`
- **Configurable Retention**: Set log retention period in days (default: 30 days)
- **Configurable Interval**: Set collection interval in seconds (default: 30 seconds)
- **Automatic Component Collection**: All core components (op-node, op-geth, op-batcher, op-proposer) are automatically collected
- **Log Download**: Download logs from running components with filtering options
- **Automatic Setup**: CloudWatch Log Groups and streams are created automatically during deployment
- **Grafana Integration**: Logs are accessible through Grafana's CloudWatch data source
- **Sidecar Architecture**: Uses AWS CLI sidecar containers for reliable log collection

**Example Log Groups Created:**
```
/aws/eks/theo0730-78s3a/op-node
/aws/eks/theo0730-78s3a/op-geth
/aws/eks/theo0730-78s3a/op-batcher
/aws/eks/theo0730-78s3a/op-proposer
```


### Supported Components
The following components can be configured for log collection:
- **op-node**: OP Node logs
- **op-geth**: OP Geth logs  
- **op-batcher**: OP Batcher logs
- **op-proposer**: OP Proposer logs


### Log Stream Structure
All components use a unified log stream name for consistent log collection:

```
CloudWatch Log Groups:
├── /aws/eks/{namespace}/op-node
│   └── sidecar-collection
├── /aws/eks/{namespace}/op-geth
│   └── sidecar-collection
├── /aws/eks/{namespace}/op-batcher
│   └── sidecar-collection
└── /aws/eks/{namespace}/op-proposer
    └── sidecar-collection
```

**Benefits of unified stream structure:**
- **Consistent Naming**: All components use the same stream name
- **Simplified Management**: Single stream per component for easy access
- **Efficient Collection**: Sidecar automatically creates and manages streams
- **Reliable Operation**: Automatic stream creation on sidecar startup


### Accessing Logs

#### Method 1: CloudWatch Logs Insights Console
1. Go to AWS CloudWatch Console
2. Navigate to Logs → Logs Insights
3. Select the log groups you want to query:
4. Enter your query and click "Run query"

#### Method 2: Grafana Dashboard
1. Access Grafana dashboard
2. Navigate to Explore
3. Select CloudWatch as the data source
4. Query logs using CloudWatch Logs Insights syntax

**Example CloudWatch Logs Insights queries:**
```sql
# Retrieve up to the 200 most recent log entries
fields @timestamp, @message |
 sort @timestamp desc |
 limit 200

# Retrieve the 50 most recent log entries containing any of the following keywords: "payload", "chain", "block", or "imported"
fields @timestamp, @message
| filter @message like /payload|chain|block|imported/
| sort @timestamp desc
| limit 50

# Retrieve up to 100 log entries within a specific timestamp range (from 1753787368000 to 1754387368000 microseconds)
fields @timestamp, @message
| filter @timestamp >= 1753787368000
| filter @timestamp <= 1754387368000
| limit 100
```

### Log Download Features
The log collection system provides powerful download capabilities:

```bash
# Download help
trh-sdk log-collection --download

# Download logs for specific component with time filter
trh-sdk log-collection --download --component op-node --hours 1

# Download logs with keyword filtering
trh-sdk log-collection --download --component op-geth --hours 2 --keyword error

# Download logs for all components
trh-sdk log-collection --download --component all --hours 24

# Download logs with minute precision
trh-sdk log-collection --download --component op-batcher --minutes 30 --keyword warning

# Download logs with multiple filters
trh-sdk log-collection --download --component op-proposer --hours 12 --keyword failed
```

**Download Options:**
- `--component`: Specify component (op-node, op-geth, op-batcher, op-proposer, all)
- `--hours`: Number of hours to look back
- `--minutes`: Number of minutes to look back  
- `--keyword`: Filter logs by keyword (case-insensitive)

**Download Features:**
- **Time-based Filtering**: Download logs from specific time periods
- **Keyword Filtering**: Filter logs by specific keywords or error messages
- **Component Selection**: Download logs from specific components or all components
- **Local Storage**: Downloaded logs are saved to local files for analysis
- **Real-time Access**: Access logs directly from running Kubernetes pods