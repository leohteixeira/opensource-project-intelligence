/** Public catalog fixtures: names, public descriptions and public source links only. */
export interface PublicProject {
  readonly id: string;
  readonly name: string;
  readonly slug: string;
  readonly description: string;
  readonly links: readonly string[];
}

export const CATALOG: readonly PublicProject[] = [
  {
    id: '732684512931872768',
    name: 'Temporal',
    slug: 'temporal',
    description: 'Durable execution platform',
    links: [
      'github.com/temporalio/temporal',
      'npmjs.com/package/@temporalio/client',
      'docs.temporal.io',
    ],
  },
  {
    id: '732684513124761600',
    name: 'Cadence',
    slug: 'cadence',
    description: 'Workflow orchestration engine',
    links: ['github.com/cadence-workflow/cadence', 'cadenceworkflow.io'],
  },
  {
    id: '732684513351254016',
    name: 'OpenTelemetry Collector',
    slug: 'otel-collector',
    description: 'Vendor-agnostic telemetry pipeline',
    links: ['github.com/open-telemetry/opentelemetry-collector', 'opentelemetry.io/docs/collector'],
  },
  {
    id: '732684513489666048',
    name: 'Conductor',
    slug: 'conductor',
    description: 'Microservice orchestration platform',
    links: ['github.com/conductor-oss/conductor'],
  },
  {
    id: '732684513598718976',
    name: 'Argo Workflows',
    slug: 'argo-workflows',
    description: 'Kubernetes-native workflow engine',
    links: ['github.com/argoproj/argo-workflows', 'argo-workflows.readthedocs.io'],
  },
  {
    id: '732684513701470208',
    name: 'Flyte',
    slug: 'flyte',
    description: 'Workflow automation for data and ML',
    links: ['github.com/flyteorg/flyte', 'pypi.org/project/flytekit'],
  },
];
