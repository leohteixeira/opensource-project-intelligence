package sourceadapter

import (
	"testing"
	"time"
)

func TestUT266GitLabAndGiteaMapEquivalentCanonicalState(t *testing.T) {
	observed := "2026-08-22T10:00:00Z"
	gitlab, err := GitLabIssues([]byte(`[{"id":42,"iid":7,"title":"Upgrade docs","description":"Same body","web_url":"https://gitlab.example/acme/repo/-/issues/7","created_at":"`+observed+`"}]`), []int64{101})
	if err != nil {
		t.Fatal(err)
	}
	gitea, err := GiteaIssues([]byte(`[{"id":42,"number":7,"title":"Upgrade docs","body":"Same body","html_url":"https://gitea.example/acme/repo/issues/7","created_at":"`+observed+`"}]`), []int64{102})
	if err != nil {
		t.Fatal(err)
	}
	if len(gitlab) != 1 || len(gitea) != 1 || gitlab[0].ExternalID != gitea[0].ExternalID ||
		gitlab[0].Title != gitea[0].Title || gitlab[0].Body != gitea[0].Body ||
		!gitlab[0].ObservedAt.Equal(gitea[0].ObservedAt) || gitlab[0].EvidenceID == gitea[0].EvidenceID {
		t.Fatalf("canonical mapping differs: gitlab=%#v gitea=%#v", gitlab, gitea)
	}
	if gitlab[0].ObservedAt != time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC) {
		t.Fatalf("timestamp changed: %s", gitlab[0].ObservedAt)
	}
}

func TestRegistryAdaptersPreserveProviderUnitAndPopulation(t *testing.T) {
	observed := time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC)
	for index, registry := range []string{"npm", "nuget", "pypi"} {
		record, err := Registry(RegistryValue{Registry: registry, Package: "example",
			URL: "https://registry.example/example", Unit: "downloads",
			Population: registry + "_public", Value: 12, ObservedAt: observed,
			EvidenceID: int64(index + 1)})
		if err != nil {
			t.Fatalf("%s mapping failed: %v", registry, err)
		}
		if record.Attributes["registry"] != registry || record.Attributes["unit"] != "downloads" ||
			record.Attributes["population"] != registry+"_public" {
			t.Fatalf("%s context changed: %#v", registry, record.Attributes)
		}
	}
}
