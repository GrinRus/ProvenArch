package artifactquality

import (
	"strings"
	"testing"

	"github.com/GrinRus/ProvenArch/internal/contracts"
)

func semanticEntity(id string) contracts.Entity {
	return contracts.Entity{ID: id, Type: "service", Name: id, Provenance: contracts.Provenance{Kind: "observation", Confidence: 0.8}}
}

func TestValidateSemanticEnvelopeRejectsDanglingGraph(t *testing.T) {
	snapshot := contracts.SemanticSnapshot{
		Entities:  []contracts.Entity{semanticEntity("svc.api")},
		Edges:     []contracts.Edge{{ID: "edge.api", Type: "calls", From: "svc.api", To: "svc.missing"}},
		Findings:  []contracts.Finding{{ID: "finding.api", Title: "gap", RelatedIDs: []string{"finding.missing"}}},
		Questions: []contracts.Question{{ID: "q.api", Text: "question", RelatedIDs: []string{"q.missing"}}},
	}
	err := ValidateSemanticEnvelope(snapshot)
	if err == nil || !strings.Contains(err.Error(), "dangling to endpoint") {
		t.Fatalf("expected dangling edge semantic issue, got %v", err)
	}
}

func TestValidateSemanticEnvelopeRejectsInvalidOwnerTeam(t *testing.T) {
	for name, owner := range map[string]string{
		"wrong prefix": "org.platform",
		"missing team": "team.platform",
	} {
		t.Run(name, func(t *testing.T) {
			snapshot := contracts.SemanticSnapshot{Entities: []contracts.Entity{{ID: "svc.api", Type: "service", OwnerTeamID: owner}, semanticEntity("team.other")}}
			if err := ValidateSemanticEnvelope(snapshot); err == nil {
				t.Fatalf("expected invalid owner team %q", owner)
			}
		})
	}
	valid := contracts.SemanticSnapshot{Entities: []contracts.Entity{{ID: "svc.api", Type: "service", OwnerTeamID: "team.platform"}, {ID: "team.platform", Type: "team", Name: "Platform"}}}
	if err := ValidateSemanticEnvelope(valid); err != nil {
		t.Fatalf("expected valid owner team, got %v", err)
	}
}

func TestValidateSemanticIDCollisionsRejectsCrossShardIdentityCollision(t *testing.T) {
	if err := ValidateSemanticIDCollisions(
		contracts.SemanticSnapshot{Entities: []contracts.Entity{semanticEntity("svc.api")}},
		contracts.SemanticSnapshot{Findings: []contracts.Finding{{ID: "svc.api", Title: "collision"}}},
	); err == nil || !strings.Contains(err.Error(), "collides") {
		t.Fatalf("expected cross-shard collision, got %v", err)
	}
}

func TestValidateSemanticIDCollisionsAllowsIdenticalShardReplayButRejectsConflictingPayload(t *testing.T) {
	first := contracts.SemanticSnapshot{Entities: []contracts.Entity{semanticEntity("svc.api")}}
	if err := ValidateSemanticIDCollisions(first, first); err != nil {
		t.Fatalf("identical shard replay should be idempotent, got %v", err)
	}
	conflicting := contracts.SemanticSnapshot{Entities: []contracts.Entity{{ID: "svc.api", Type: "service", Name: "different"}}}
	if err := ValidateSemanticIDCollisions(first, conflicting); err == nil || !strings.Contains(err.Error(), "collides") {
		t.Fatalf("expected conflicting same-kind payload rejection, got %v", err)
	}
}

func TestValidateSemanticIDCollisionsAllowsSameRepoEntityObservations(t *testing.T) {
	left := contracts.Entity{
		ID:   "svc.bank.accounts-db",
		Type: "database",
		Name: "accounts-db",
		Provenance: contracts.Provenance{
			Kind:     "observation",
			Evidence: []contracts.Evidence{{Repo: "bank-of-anthos", Path: "README.md"}},
		},
	}
	right := contracts.Entity{
		ID:   "svc.bank.accounts-db",
		Type: "datastore",
		Name: "Accounts PostgreSQL database",
		Provenance: contracts.Provenance{
			Kind:     "observation",
			Evidence: []contracts.Evidence{{Repo: "bank-of-anthos", Path: "src/accounts/accounts-db/README.md"}},
		},
	}
	if err := ValidateSemanticIDCollisions(
		contracts.SemanticSnapshot{Entities: []contracts.Entity{left}},
		contracts.SemanticSnapshot{Entities: []contracts.Entity{right}},
	); err != nil {
		t.Fatalf("same-repo exact entity observations should merge during normalization, got %v", err)
	}
}

func TestValidateSemanticIDCollisionsAllowsSvcInfrastructureServiceAlias(t *testing.T) {
	evidence := func(path string) contracts.Provenance {
		return contracts.Provenance{Kind: "observation", Evidence: []contracts.Evidence{{Repo: "posthog", Path: path}}}
	}
	left := contracts.Entity{
		ID:         "svc.posthog.temporal",
		Type:       "infrastructure",
		Name:       "Temporal workflow runtime",
		Provenance: evidence("docker-compose.dev.yml"),
	}
	right := contracts.Entity{
		ID:         "svc.posthog.temporal",
		Type:       "service",
		Name:       "Temporal dynamic configuration surface",
		Provenance: evidence("docker/temporal/dynamicconfig/README.md"),
	}
	if err := ValidateSemanticIDCollisions(
		contracts.SemanticSnapshot{Entities: []contracts.Entity{left, right}},
	); err != nil {
		t.Fatalf("svc infrastructure/service aliases should merge, got %v", err)
	}
	crossRepo := right
	crossRepo.Provenance = evidence("../other-repo/temporal.md")
	crossRepo.Provenance.Evidence[0].Repo = "other-repo"
	if err := ValidateSemanticIDCollisions(
		contracts.SemanticSnapshot{Entities: []contracts.Entity{left}},
		contracts.SemanticSnapshot{Entities: []contracts.Entity{crossRepo}},
	); err == nil || !strings.Contains(err.Error(), "collides") {
		t.Fatalf("svc infrastructure/service aliases across repositories should remain a collision, got %v", err)
	}
}

func TestValidateSemanticIDCollisionsAllowsComponentServiceAlias(t *testing.T) {
	evidence := func(path string) contracts.Provenance {
		return contracts.Provenance{Kind: "observation", Evidence: []contracts.Evidence{{Repo: "posthog", Path: path}}}
	}
	left := contracts.Entity{
		ID:         "component.posthog.web",
		Type:       "component",
		Name:       "Web application",
		Provenance: evidence("docker-compose.dev-full.yml"),
	}
	right := contracts.Entity{
		ID:         "component.posthog.web",
		Type:       "service",
		Name:       "PostHog web application",
		Provenance: evidence("docker-compose.dev-full.yml"),
	}
	if err := ValidateSemanticIDCollisions(contracts.SemanticSnapshot{Entities: []contracts.Entity{left, right}}); err != nil {
		t.Fatalf("component/service aliases should merge, got %v", err)
	}
	crossRepo := right
	crossRepo.Provenance = evidence("../other-repo/web.md")
	crossRepo.Provenance.Evidence[0].Repo = "other-repo"
	if err := ValidateSemanticIDCollisions(
		contracts.SemanticSnapshot{Entities: []contracts.Entity{left}},
		contracts.SemanticSnapshot{Entities: []contracts.Entity{crossRepo}},
	); err == nil || !strings.Contains(err.Error(), "collides") {
		t.Fatalf("component/service aliases across repositories should remain a collision, got %v", err)
	}
}

func TestValidateSemanticIDCollisionsRejectsSameRepoUnrelatedEntityObservation(t *testing.T) {
	left := contracts.Entity{
		ID:   "svc.bank.accounts-db",
		Type: "service",
		Name: "Accounts",
		Provenance: contracts.Provenance{
			Kind:     "observation",
			Evidence: []contracts.Evidence{{Repo: "bank-of-anthos", Path: "README.md"}},
		},
	}
	right := left
	right.Name = "Payments API"
	if err := ValidateSemanticIDCollisions(
		contracts.SemanticSnapshot{Entities: []contracts.Entity{left}},
		contracts.SemanticSnapshot{Entities: []contracts.Entity{right}},
	); err == nil || !strings.Contains(err.Error(), "collides") {
		t.Fatalf("unrelated same-repo exact ID should remain a collision, got %v", err)
	}
}

func TestValidateSemanticIDCollisionsAllowsSameRepoWeakEdgeIDRekey(t *testing.T) {
	left := contracts.Edge{
		ID:   "edge.balance-reader.reads-ledger-db",
		Type: "reads-from",
		From: "service.balance-reader",
		To:   "service.ledger-db",
		Provenance: contracts.Provenance{
			Kind:     "observation",
			Evidence: []contracts.Evidence{{Repo: "bank-of-anthos", Path: "README.md"}},
		},
	}
	right := left
	right.From = "svc.bank.of.anthos.src.ledger.balancereader"
	right.To = "db.bank.of.anthos.ledger-db"
	right.Provenance.Evidence = []contracts.Evidence{{Repo: "bank-of-anthos", Path: "src/ledger/balancereader/README.md"}}
	if err := ValidateSemanticIDCollisions(
		contracts.SemanticSnapshot{Edges: []contracts.Edge{left}},
		contracts.SemanticSnapshot{Edges: []contracts.Edge{right}},
	); err != nil {
		t.Fatalf("same-repo weak edge IDs should be rekeyed during normalization, got %v", err)
	}
}

func TestValidateSemanticIDCollisionsAllowsRouteEdgeTypeAliases(t *testing.T) {
	evidence := func(path string) contracts.Provenance {
		return contracts.Provenance{
			Kind:     "observation",
			Evidence: []contracts.Evidence{{Repo: "posthog", Path: path}},
		}
	}
	left := contracts.Edge{
		ID:         "edge.posthog.proxy.routes-web",
		Type:       "routes",
		From:       "component.posthog.proxy",
		To:         "service.posthog.web",
		Provenance: evidence("docker-compose.base.yml"),
	}
	right := contracts.Edge{
		ID:         "edge.posthog.proxy.routes-web",
		Type:       "routes_to",
		From:       "component.posthog.proxy",
		To:         "component.posthog.web",
		Provenance: evidence("docker-compose.dev.yml"),
	}
	if err := ValidateSemanticIDCollisions(
		contracts.SemanticSnapshot{Edges: []contracts.Edge{left}},
		contracts.SemanticSnapshot{Edges: []contracts.Edge{right}},
	); err != nil {
		t.Fatalf("same-repo route relation aliases should be rekeyed during normalization, got %v", err)
	}
}

func TestValidateSemanticIDCollisionsRejectsUnrelatedEdgeType(t *testing.T) {
	evidence := func(path string) contracts.Provenance {
		return contracts.Provenance{
			Kind:     "observation",
			Evidence: []contracts.Evidence{{Repo: "posthog", Path: path}},
		}
	}
	left := contracts.Edge{
		ID:         "edge.posthog.proxy.routes-web",
		Type:       "routes",
		From:       "component.posthog.proxy",
		To:         "service.posthog.web",
		Provenance: evidence("docker-compose.base.yml"),
	}
	right := left
	right.Type = "calls"
	right.Provenance = evidence("docker-compose.dev.yml")
	if err := ValidateSemanticIDCollisions(
		contracts.SemanticSnapshot{Edges: []contracts.Edge{left}},
		contracts.SemanticSnapshot{Edges: []contracts.Edge{right}},
	); err == nil || !strings.Contains(err.Error(), "collides") {
		t.Fatalf("unrelated edge relation types should remain a collision, got %v", err)
	}
}

func TestValidateSemanticIDCollisionsAllowsCanonicalIDTypeFamilies(t *testing.T) {
	evidence := func(path string) contracts.Provenance {
		return contracts.Provenance{Kind: "observation", Evidence: []contracts.Evidence{{Repo: "bank-of-anthos", Path: path}}}
	}
	observations := []contracts.SemanticSnapshot{
		{Entities: []contracts.Entity{{ID: "svc.bank.of.anthos", Type: "service", Name: "Bank of Anthos", Provenance: evidence("README.md")}}},
		{Entities: []contracts.Entity{{ID: "svc.bank.of.anthos", Type: "application", Name: "Bank of Anthos", Provenance: evidence("README.md")}}},
		{Entities: []contracts.Entity{{ID: "svc.bank.of.anthos", Type: "platform", Name: "Bank of Anthos", Provenance: evidence("README.md")}}},
		{Entities: []contracts.Entity{{ID: "svc.bank.of.anthos", Type: "service-platform", Name: "Bank of Anthos platform", Provenance: evidence("docker-compose.yml")}}},
		{Entities: []contracts.Entity{{ID: "svc.bank.of.anthos", Type: "application-service", Name: "Bank of Anthos application service", Provenance: evidence("settings.gradle")}}},
		{Entities: []contracts.Entity{{ID: "svc.bank.of.anthos", Type: "component", Name: "Bank of Anthos", Provenance: evidence("package.json")}}},
		{Entities: []contracts.Entity{{ID: "svc.bank.of.anthos", Type: "application-component", Name: "Bank of Anthos", Provenance: evidence("playwright/package.json")}}},
		{Entities: []contracts.Entity{{ID: "svc.bank.of.anthos", Type: "service-group", Name: "Bank of Anthos", Provenance: evidence("README.adoc")}}},
		{Entities: []contracts.Entity{{ID: "svc.bank.of.anthos", Type: "repository", Name: "Bank of Anthos repository", Provenance: evidence("README.md")}}},
		{Entities: []contracts.Entity{{ID: "svc.bank.of.anthos", Type: "api-gateway", Name: "Bank of Anthos API gateway", Provenance: evidence("gateway/README.md")}}},
		{Entities: []contracts.Entity{{ID: "svc.bank.of.anthos", Type: "service-suite", Name: "Bank of Anthos service suite", Provenance: evidence("services/README.md")}}},
		{Entities: []contracts.Entity{{ID: "svc.bank.of.anthos", Type: "service-landscape", Name: "Bank of Anthos service landscape", Provenance: evidence("docs/architecture.md")}}},
		{Entities: []contracts.Entity{{ID: "svc.bank.of.anthos", Type: "service-system", Name: "Bank of Anthos service system", Provenance: evidence("docs/system.md")}}},
		{Entities: []contracts.Entity{{ID: "svc.bank.of.anthos", Type: "data-service", Name: "Bank of Anthos data service", Provenance: evidence("data/README.md")}}},
		{Entities: []contracts.Entity{{ID: "svc.bank.of.anthos.balance-reader", Type: "dependency", Name: "Balance Reader", Provenance: evidence("src/ledger/ledgerwriter/README.md")}}},
		{Entities: []contracts.Entity{{ID: "svc.bank.of.anthos.balance-reader", Type: "service", Name: "Balance Reader", Provenance: evidence("src/ledger/balancereader/README.md")}}},
		{Entities: []contracts.Entity{{ID: "svc.bank.of.anthos", Type: "domain", Name: "Bank of Anthos", Provenance: evidence("docs/README.md")}}},
		{Entities: []contracts.Entity{{ID: "team.bank.of.anthos.default-security", Type: "owner-team", Name: "security", Provenance: evidence(".github/CODEOWNERS")}}},
		{Entities: []contracts.Entity{{ID: "team.bank.of.anthos.default-security", Type: "review-owner", Name: "security", Provenance: evidence(".github/CODEOWNERS")}}},
		{Entities: []contracts.Entity{{ID: "team.bank.of.anthos.default-security", Type: "review-team", Name: "security team", Provenance: evidence(".github/CODEOWNERS")}}},
		{Entities: []contracts.Entity{{ID: "team.bank.of.anthos.default-security", Type: "approval-owner", Name: "security", Provenance: evidence(".github/CODEOWNERS")}}},
		{Entities: []contracts.Entity{{ID: "svc.ftgo.application", Type: "service-domain", Name: "FTGO application", Provenance: evidence("README.adoc")}}},
		{Entities: []contracts.Entity{{ID: "svc.bank.of.anthos", Type: "application-surface", Name: "Bank of Anthos", Provenance: evidence("package.json")}}},
		{Entities: []contracts.Entity{{ID: "system.bank.of.anthos", Type: "system", Name: "Bank of Anthos application", Provenance: evidence("README.md")}}},
		{Entities: []contracts.Entity{{ID: "system.bank.of.anthos", Type: "application", Name: "Bank of Anthos", Provenance: evidence("README.md")}}},
		{Entities: []contracts.Entity{{ID: "db.bank.of.anthos.accounts", Type: "stateful-workload", Name: "accounts-db PostgreSQL StatefulSet", Provenance: evidence("kubernetes-manifests/accounts-db.yaml")}}},
		{Entities: []contracts.Entity{{ID: "db.bank.of.anthos.accounts", Type: "datastore", Name: "Accounts database", Provenance: evidence("README.md")}}},
		{Entities: []contracts.Entity{{ID: "datastore.posthog.objectstorage", Type: "datastore", Name: "S3-compatible Ducklake object store", Provenance: evidence("devenv/duckgres.yaml")}}},
		{Entities: []contracts.Entity{{ID: "datastore.posthog.objectstorage", Type: "datastore", Name: "PostHog object storage service", Provenance: evidence("docker-compose.base.yml")}}},
		{Entities: []contracts.Entity{{ID: "datastore.posthog.clickhouse", Type: "datastore", Name: "ClickHouse analytics store", Provenance: evidence("docker-compose.base.yml")}}},
		{Entities: []contracts.Entity{{ID: "datastore.posthog.clickhouse", Type: "datastore", Name: "Session recording metadata store", Provenance: evidence("nodejs/src/session-recording/README.md")}}},
		{Entities: []contracts.Entity{{ID: "datastore.posthog.clickhouse", Type: "datastore", Name: "ClickHouse preaggregation tables", Provenance: evidence("products/analytics_platform/backend/lazy_computation/README.md")}}},
		{Entities: []contracts.Entity{{ID: "team.bank.of.anthos.default-maintainers", Type: "team", Name: "maintainers", Provenance: evidence(".github/CODEOWNERS")}}},
		{Entities: []contracts.Entity{{ID: "team.bank.of.anthos.default-maintainers", Type: "owner-group", Name: "GoogleCloudPlatform maintainers", Provenance: evidence(".github/CODEOWNERS")}}},
		{Entities: []contracts.Entity{{ID: "team.bank.of.anthos.default-owners", Type: "repository-owners", Name: "GoogleCloudPlatform maintainers", Provenance: evidence(".github/CODEOWNERS")}}},
		{Entities: []contracts.Entity{{ID: "team.bank.of.anthos.default-owners", Type: "team", Name: "GoogleCloudPlatform maintainers", Provenance: evidence(".github/CODEOWNERS")}}},
		{Entities: []contracts.Entity{{ID: "infra.bank.of.anthos.gke", Type: "runtime-platform", Name: "Google Kubernetes Engine", Provenance: evidence("README.md")}}},
		{Entities: []contracts.Entity{{ID: "infra.bank.of.anthos.gke", Type: "infrastructure", Name: "Google Kubernetes Engine infrastructure", Provenance: evidence("iac/tf-anthos-gke/README.md")}}},
		{Entities: []contracts.Entity{{ID: "infra.bank.of.anthos.kafka", Type: "message-broker", Name: "Kafka broker", Provenance: evidence("docker-compose.yml")}}},
		{Entities: []contracts.Entity{{ID: "infra.bank.of.anthos.kafka", Type: "messaging-infrastructure", Name: "Kafka messaging infrastructure", Provenance: evidence("kubernetes/kafka.yaml")}}},
		{Entities: []contracts.Entity{{ID: "infra.bank.of.anthos.mysql", Type: "datastore", Name: "MySQL database", Provenance: evidence("docker-compose.yml")}}},
		{Entities: []contracts.Entity{{ID: "infra.bank.of.anthos.mysql", Type: "database-infrastructure", Name: "MySQL database infrastructure", Provenance: evidence("deploy/mysql.yaml")}}},
		{Entities: []contracts.Entity{{ID: "infra.bank.of.anthos.zookeeper", Type: "coordination-service", Name: "ZooKeeper", Provenance: evidence("docker-compose.yml")}}},
		{Entities: []contracts.Entity{{ID: "infra.bank.of.anthos.zookeeper", Type: "coordination-infrastructure", Name: "ZooKeeper coordination infrastructure", Provenance: evidence("kubernetes/zookeeper.yaml")}}},
		{Entities: []contracts.Entity{{ID: "infra.ftgo.cdc", Type: "infrastructure", Name: "Eventuate CDC service", Provenance: evidence("docker-compose.yml")}}},
		{Entities: []contracts.Entity{{ID: "infra.ftgo.cdc", Type: "change-data-capture-service", Name: "Eventuate CDC service", Provenance: evidence("docker-compose.yml")}}},
	}
	if err := ValidateSemanticIDCollisions(observations...); err != nil {
		t.Fatalf("canonical ID type families should merge, got %v", err)
	}
}

func TestValidateSemanticIDCollisionsAllowsServicePrefixAliases(t *testing.T) {
	evidence := func(path string) contracts.Provenance {
		return contracts.Provenance{Kind: "observation", Evidence: []contracts.Evidence{{Repo: "ftgo-application", Path: path}}}
	}
	observations := []contracts.SemanticSnapshot{
		{Entities: []contracts.Entity{{ID: "service.ftgo.application", Type: "service-system", Name: "FTGO example application", Provenance: evidence("README.adoc")}}},
		{Entities: []contracts.Entity{{ID: "service.ftgo.application", Type: "application", Name: "FTGO application", Provenance: evidence("skaffold.yaml")}}},
		{Entities: []contracts.Entity{{ID: "service.ftgo.application", Type: "service-landscape", Name: "FTGO example microservice application", Provenance: evidence("README.adoc")}}},
		{Entities: []contracts.Entity{{ID: "svc.ftgo.api-gateway", Type: "service", Name: "API Gateway", Provenance: evidence("README.adoc")}}},
		{Entities: []contracts.Entity{{ID: "svc.ftgo.api-gateway", Type: "gateway", Name: "API Gateway", Provenance: evidence("docker-compose.yml")}}},
	}
	if err := ValidateSemanticIDCollisions(observations...); err != nil {
		t.Fatalf("same-repo service-prefix aliases should merge, got %v", err)
	}
}

func TestValidateSemanticIDCollisionsAllowsTechTechnologyFrameworkAliases(t *testing.T) {
	evidence := func(path string) contracts.Provenance {
		return contracts.Provenance{Kind: "observation", Evidence: []contracts.Evidence{{Repo: "posthog", Path: path}}}
	}
	left := contracts.Entity{ID: "tech.django", Type: "technology", Name: "Django", Provenance: evidence("products/README.md")}
	right := contracts.Entity{ID: "tech.django", Type: "framework", Name: "Django", Provenance: evidence("pyproject.toml")}
	if err := ValidateSemanticIDCollisions(contracts.SemanticSnapshot{Entities: []contracts.Entity{left}}, contracts.SemanticSnapshot{Entities: []contracts.Entity{right}}); err != nil {
		t.Fatalf("same-repo tech technology/framework aliases should merge, got %v", err)
	}
}

func TestValidateSemanticIDCollisionsAllowsSvcDatastoreServiceAlias(t *testing.T) {
	evidence := func(path string) contracts.Provenance {
		return contracts.Provenance{Kind: "observation", Evidence: []contracts.Evidence{{Repo: "posthog", Path: path}}}
	}
	left := contracts.Entity{
		ID:         "svc.posthog.clickhouse",
		Type:       "datastore",
		Name:       "ClickHouse analytics store",
		Provenance: evidence("docker-compose.base.yml"),
	}
	right := left
	right.Type = "service"
	right.Provenance = evidence("docker-compose.dev-full.yml")
	if err := ValidateSemanticIDCollisions(
		contracts.SemanticSnapshot{Entities: []contracts.Entity{left}},
		contracts.SemanticSnapshot{Entities: []contracts.Entity{right}},
	); err != nil {
		t.Fatalf("same-repo svc datastore/service aliases should merge, got %v", err)
	}
}

func TestValidateSemanticIDCollisionsAllowsSvcClickhouseAnalyticalDatabaseAlias(t *testing.T) {
	evidence := func(path string) contracts.Provenance {
		return contracts.Provenance{Kind: "observation", Evidence: []contracts.Evidence{{Repo: "posthog", Path: path}}}
	}
	left := contracts.Entity{
		ID:         "svc.posthog.clickhouse",
		Type:       "analytical-database",
		Name:       "Local ClickHouse analytical database",
		Provenance: evidence("devenv/README.md"),
	}
	right := contracts.Entity{
		ID:         "svc.posthog.clickhouse",
		Type:       "service",
		Name:       "PostHog ClickHouse migration and topology surface",
		Provenance: evidence("clickhouse/migrations/README.md"),
	}
	if err := ValidateSemanticIDCollisions(
		contracts.SemanticSnapshot{Entities: []contracts.Entity{left}},
		contracts.SemanticSnapshot{Entities: []contracts.Entity{right}},
	); err != nil {
		t.Fatalf("same-repo svc clickhouse analytical-database/service aliases should merge, got %v", err)
	}
}

func TestValidateSemanticIDCollisionsAllowsRuntimeDeploymentTopologyAlias(t *testing.T) {
	evidence := func(path string) contracts.Provenance {
		return contracts.Provenance{Kind: "observation", Evidence: []contracts.Evidence{{Repo: "posthog", Path: path}}}
	}
	left := contracts.Entity{
		ID:         "runtime.posthog.compose",
		Type:       "runtime",
		Name:       "PostHog Docker Compose local/hobby runtime",
		Provenance: evidence("docker-compose.base.yml"),
	}
	right := contracts.Entity{
		ID:         "runtime.posthog.compose",
		Type:       "deployment-topology",
		Name:       "PostHog base Compose runtime",
		Provenance: evidence("docker-compose.dev.yml"),
	}
	if err := ValidateSemanticIDCollisions(
		contracts.SemanticSnapshot{Entities: []contracts.Entity{left}},
		contracts.SemanticSnapshot{Entities: []contracts.Entity{right}},
	); err != nil {
		t.Fatalf("same-repo runtime/deployment-topology aliases should merge, got %v", err)
	}
}

func TestValidateSemanticIDCollisionsRejectsRuntimeAliasAcrossRepos(t *testing.T) {
	left := contracts.Entity{
		ID:   "runtime.posthog.compose",
		Type: "runtime",
		Name: "PostHog Docker Compose runtime",
		Provenance: contracts.Provenance{
			Kind:     "observation",
			Evidence: []contracts.Evidence{{Repo: "posthog", Path: "docker-compose.base.yml"}},
		},
	}
	right := left
	right.Type = "deployment-topology"
	right.Provenance.Evidence = []contracts.Evidence{{Repo: "other-repo", Path: "docker-compose.yml"}}
	if err := ValidateSemanticIDCollisions(
		contracts.SemanticSnapshot{Entities: []contracts.Entity{left}},
		contracts.SemanticSnapshot{Entities: []contracts.Entity{right}},
	); err == nil || !strings.Contains(err.Error(), "collides") {
		t.Fatalf("runtime aliases across repositories should remain a collision, got %v", err)
	}
}

func TestValidateSemanticIDCollisionsAllowsCaptureLogsNameOrderAlias(t *testing.T) {
	evidence := func(path string) contracts.Provenance {
		return contracts.Provenance{Kind: "observation", Evidence: []contracts.Evidence{{Repo: "posthog", Path: path}}}
	}
	left := contracts.Entity{
		ID:         "svc.posthog.capture-logs",
		Type:       "service",
		Name:       "Capture logs and traces service",
		Provenance: evidence("docker-compose.base.yml"),
	}
	right := left
	right.Name = "PostHog OTLP log capture service"
	right.Provenance = evidence("rust/capture-logs/README.md")
	if err := ValidateSemanticIDCollisions(
		contracts.SemanticSnapshot{Entities: []contracts.Entity{left}},
		contracts.SemanticSnapshot{Entities: []contracts.Entity{right}},
	); err != nil {
		t.Fatalf("same-repo capture-logs name-order aliases should merge, got %v", err)
	}
}

func TestValidateSemanticIDCollisionsRejectsUnrelatedDatastoreName(t *testing.T) {
	evidence := func(path string) contracts.Provenance {
		return contracts.Provenance{Kind: "observation", Evidence: []contracts.Evidence{{Repo: "posthog", Path: path}}}
	}
	left := contracts.SemanticSnapshot{Entities: []contracts.Entity{{ID: "datastore.posthog.clickhouse", Type: "datastore", Name: "ClickHouse", Provenance: evidence("docker-compose.base.yml")}}}
	right := contracts.SemanticSnapshot{Entities: []contracts.Entity{{ID: "datastore.posthog.clickhouse", Type: "datastore", Name: "Kafka broker", Provenance: evidence("docker-compose.dev.yml")}}}
	if err := ValidateSemanticIDCollisions(left, right); err == nil || !strings.Contains(err.Error(), "collides") {
		t.Fatalf("unrelated datastore names should remain a collision, got %v", err)
	}
}

func TestValidateSemanticIDCollisionsAllowsExternalSystemProductAliases(t *testing.T) {
	evidence := func(path string) contracts.Provenance {
		return contracts.Provenance{Kind: "observation", Evidence: []contracts.Evidence{{Repo: "bank-of-anthos", Path: path}}}
	}
	observations := []contracts.SemanticSnapshot{
		{Entities: []contracts.Entity{{ID: "external.system.bank.of.anthos.gke", Type: "external.system", Name: "Google Kubernetes Engine", Provenance: evidence("docs/ci-cd-pipeline.md")}}},
		{Entities: []contracts.Entity{{ID: "external.system.bank.of.anthos.gke", Type: "external.system", Name: "Google Cloud GKE and Anthos platform", Provenance: evidence("iac/tf-anthos-gke/README.md")}}},
	}
	if err := ValidateSemanticIDCollisions(observations...); err != nil {
		t.Fatalf("external-system product name/acronym aliases should merge, got %v", err)
	}
	conflicting := observations[1]
	conflicting.Entities[0].Name = "Cloud SQL"
	if err := ValidateSemanticIDCollisions(observations[0], conflicting); err == nil || !strings.Contains(err.Error(), "collides") {
		t.Fatalf("unrelated external-system names should remain a collision, got %v", err)
	}
}

func TestValidateSemanticEnvelopeJSONRejectsUnknownNestedFields(t *testing.T) {
	raw := []byte(`{"semantic":{"coverage":{"observed":[],"missing":[],"notes":[]},"questions":[],"entities":[{"id":"svc.api","type":"service","name":"API","provenance":{"kind":"observation","confidence":0.8,"unexpected":true}}],"edges":[],"findings":[]}}`)
	if err := ValidateSemanticEnvelopeJSON(raw); err == nil || !strings.Contains(err.Error(), `unknown field "unexpected"`) {
		t.Fatalf("expected unknown semantic field rejection, got %v", err)
	}
}
