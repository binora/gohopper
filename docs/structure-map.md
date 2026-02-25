# GraphHopper -> GoHopper Structure Map

This map keeps contributor mental models aligned with GraphHopper 11.0.

| GraphHopper (Java) | GoHopper (Go) |
|---|---|
| `core/.../GraphHopperConfig.java` | `core/graphhopper_config.go` |
| `core/.../GraphHopper.java` | `core/graphhopper.go` |
| `core/.../routing/Router.java` | `core/routing/router.go` |
| `core/.../routing/ch/CHPreparationHandler.java` | `core/routing/ch/preparation_handler.go` |
| `core/.../routing/lm/LMPreparationHandler.java` | `core/routing/lm/preparation_handler.go` |
| `core/.../routing/weighting/custom/CustomModelParser.java` | `core/routing/weighting/custom/custom_model_parser.go` |
| `core/.../storage/BaseGraph.java` | `core/storage/base_graph.go` |
| `core/.../storage/index/LocationIndex*` | `core/storage/index/location_index.go` |
| `web-api/.../GHRequest.java` | `web-api/gh_request.go` |
| `web-api/.../GHResponse.java` | `web-api/gh_response.go` |
| `web-bundle/.../resources/RouteResource.java` | `web-bundle/resources/route_resource.go` |
| `web-bundle/.../resources/NearestResource.java` | `web-bundle/resources/nearest_resource.go` |
| `web-bundle/.../resources/InfoResource.java` | `web-bundle/resources/info_resource.go` |
| `web-bundle/.../resources/HealthCheckResource.java` | `web-bundle/resources/health_check_resource.go` |

