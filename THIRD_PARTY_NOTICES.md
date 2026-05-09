# Third-Party Notices

Generated on 2026-05-09 from the current dependency lock state.

This file summarizes third-party dependencies used by AI Proxy release artifacts. It is not legal advice. Preserve this file together with distributed binaries and container images, along with the root LICENSE file.

## Coverage

- Go modules are collected with `go list -m -json all` from `core`, which includes the workspace modules used by the backend release build.
- Web production dependencies are collected with `pnpm licenses list --prod --json` from `web`.
- Go license identifiers are best-effort detections from module LICENSE/COPYING/NOTICE files in the local module cache. Entries marked `UNKNOWN` need manual review before a formal compliance sign-off.

## License Summary

### Go Modules

| License | Count |
| --- | --- |
| Apache-2.0 | 105 |
| BSD | 15 |
| BSD-3-Clause | 58 |
| ISC | 1 |
| MIT | 105 |
| MPL-2.0 | 2 |
| UNKNOWN | 23 |

### Web Production Dependencies

| License | Count |
| --- | --- |
| 0BSD | 1 |
| Apache-2.0 | 5 |
| BlueOak-1.0.0 | 2 |
| BSD-2-Clause | 1 |
| BSD-3-Clause | 5 |
| CC0-1.0 | 1 |
| ISC | 7 |
| MIT | 282 |
| MPL-2.0 | 2 |

## Go Modules

| Module | Version | Detected License | Source |
| --- | --- | --- | --- |
| cel.dev/expr | v0.25.1 | Apache-2.0 |  |
| cloud.google.com/go | v0.123.0 | Apache-2.0 |  |
| cloud.google.com/go/auth | v0.19.0 | Apache-2.0 |  |
| cloud.google.com/go/auth/oauth2adapt | v0.2.8 | Apache-2.0 |  |
| cloud.google.com/go/compute/metadata | v0.9.0 | Apache-2.0 |  |
| cloud.google.com/go/iam | v1.6.0 | Apache-2.0 |  |
| cloud.google.com/go/longrunning | v0.8.0 | UNKNOWN |  |
| cloud.google.com/go/translate | v1.10.3 | Apache-2.0 |  |
| dario.cat/mergo | v1.0.2 | BSD-3-Clause |  |
| filippo.io/edwards25519 | v1.2.0 | BSD-3-Clause |  |
| github.com/AdaLogics/go-fuzz-headers | v0.0.0-20240806141605-e8a1dd7889d6 | Apache-2.0 | https://github.com/AdaLogics/go-fuzz-headers |
| github.com/andybalholm/cascadia | v1.3.3 | BSD | https://github.com/andybalholm/cascadia |
| github.com/araddon/dateparse | v0.0.0-20210429162001-6b43995a97de | MIT | https://github.com/araddon/dateparse |
| github.com/aws/aws-sdk-go-v2 | v1.41.4 | Apache-2.0 | https://github.com/aws/aws-sdk-go-v2 |
| github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream | v1.7.8 | Apache-2.0 | https://github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream |
| github.com/aws/aws-sdk-go-v2/credentials | v1.19.12 | Apache-2.0 | https://github.com/aws/aws-sdk-go-v2/credentials |
| github.com/aws/aws-sdk-go-v2/feature/ec2/imds | v1.18.20 | UNKNOWN | https://github.com/aws/aws-sdk-go-v2/feature/ec2/imds |
| github.com/aws/aws-sdk-go-v2/internal/configsources | v1.4.20 | Apache-2.0 | https://github.com/aws/aws-sdk-go-v2/internal/configsources |
| github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 | v2.7.20 | Apache-2.0 | https://github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 |
| github.com/aws/aws-sdk-go-v2/service/bedrockruntime | v1.50.3 | Apache-2.0 | https://github.com/aws/aws-sdk-go-v2/service/bedrockruntime |
| github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding | v1.13.7 | UNKNOWN | https://github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding |
| github.com/aws/aws-sdk-go-v2/service/internal/presigned-url | v1.13.20 | UNKNOWN | https://github.com/aws/aws-sdk-go-v2/service/internal/presigned-url |
| github.com/aws/aws-sdk-go-v2/service/signin | v1.0.8 | UNKNOWN | https://github.com/aws/aws-sdk-go-v2/service/signin |
| github.com/aws/aws-sdk-go-v2/service/sso | v1.30.13 | UNKNOWN | https://github.com/aws/aws-sdk-go-v2/service/sso |
| github.com/aws/aws-sdk-go-v2/service/ssooidc | v1.35.17 | UNKNOWN | https://github.com/aws/aws-sdk-go-v2/service/ssooidc |
| github.com/aws/aws-sdk-go-v2/service/sts | v1.41.9 | UNKNOWN | https://github.com/aws/aws-sdk-go-v2/service/sts |
| github.com/aws/smithy-go | v1.24.2 | Apache-2.0 | https://github.com/aws/smithy-go |
| github.com/Azure/go-ansiterm | v0.0.0-20210617225240-d185dfc1b5a1 | MIT | https://github.com/Azure/go-ansiterm |
| github.com/bsm/ginkgo/v2 | v2.12.0 | MIT | https://github.com/bsm/ginkgo/v2 |
| github.com/bsm/gomega | v1.27.10 | MIT | https://github.com/bsm/gomega |
| github.com/bytedance/gopkg | v0.1.4 | Apache-2.0 | https://github.com/bytedance/gopkg |
| github.com/bytedance/sonic | v1.15.0 | Apache-2.0 | https://github.com/bytedance/sonic |
| github.com/bytedance/sonic/loader | v0.5.0 | Apache-2.0 | https://github.com/bytedance/sonic/loader |
| github.com/cenkalti/backoff/v4 | v4.2.1 | MIT | https://github.com/cenkalti/backoff/v4 |
| github.com/cespare/xxhash/v2 | v2.3.0 | MIT | https://github.com/cespare/xxhash/v2 |
| github.com/chenzhuoyu/base64x | v0.0.0-20221115062448-fe3a3abad311 | Apache-2.0 | https://github.com/chenzhuoyu/base64x |
| github.com/cloudwego/base64x | v0.1.6 | Apache-2.0 | https://github.com/cloudwego/base64x |
| github.com/cncf/xds/go | v0.0.0-20251210132809-ee656c7534f5 | Apache-2.0 | https://github.com/cncf/xds/go |
| github.com/containerd/errdefs | v1.0.0 | Apache-2.0 | https://github.com/containerd/errdefs |
| github.com/containerd/errdefs/pkg | v0.3.0 | Apache-2.0 | https://github.com/containerd/errdefs/pkg |
| github.com/containerd/log | v0.1.0 | Apache-2.0 | https://github.com/containerd/log |
| github.com/containerd/platforms | v0.2.1 | Apache-2.0 | https://github.com/containerd/platforms |
| github.com/containerd/typeurl/v2 | v2.2.0 | UNKNOWN | https://github.com/containerd/typeurl/v2 |
| github.com/cpuguy83/dockercfg | v0.3.2 | MIT | https://github.com/cpuguy83/dockercfg |
| github.com/cpuguy83/go-md2man/v2 | v2.0.0-20190314233015-f79a8a8ca69d | MIT | https://github.com/cpuguy83/go-md2man/v2 |
| github.com/creack/pty | v1.1.18 | MIT | https://github.com/creack/pty |
| github.com/davecgh/go-spew | v1.1.1 | ISC | https://github.com/davecgh/go-spew |
| github.com/dgryski/go-rendezvous | v0.0.0-20200823014737-9f7001d12a5f | MIT | https://github.com/dgryski/go-rendezvous |
| github.com/distribution/reference | v0.6.0 | Apache-2.0 | https://github.com/distribution/reference |
| github.com/dlclark/regexp2 | v1.11.5 | MIT | https://github.com/dlclark/regexp2 |
| github.com/dlclark/regexp2cg | v0.2.0 | MIT | https://github.com/dlclark/regexp2cg |
| github.com/docker/docker | v28.3.3+incompatible | Apache-2.0 | https://github.com/docker/docker |
| github.com/docker/go-connections | v0.6.0 | Apache-2.0 | https://github.com/docker/go-connections |
| github.com/docker/go-units | v0.5.0 | Apache-2.0 | https://github.com/docker/go-units |
| github.com/dustin/go-humanize | v1.0.1 | MIT | https://github.com/dustin/go-humanize |
| github.com/ebitengine/purego | v0.8.4 | Apache-2.0 | https://github.com/ebitengine/purego |
| github.com/envoyproxy/go-control-plane | v0.14.0 | Apache-2.0 | https://github.com/envoyproxy/go-control-plane |
| github.com/envoyproxy/go-control-plane/envoy | v1.36.0 | Apache-2.0 | https://github.com/envoyproxy/go-control-plane/envoy |
| github.com/envoyproxy/go-control-plane/ratelimit | v0.1.0 | Apache-2.0 | https://github.com/envoyproxy/go-control-plane/ratelimit |
| github.com/envoyproxy/protoc-gen-validate | v1.3.0 | Apache-2.0 | https://github.com/envoyproxy/protoc-gen-validate |
| github.com/felixge/httpsnoop | v1.0.4 | MIT | https://github.com/felixge/httpsnoop |
| github.com/frankban/quicktest | v1.14.6 | MIT | https://github.com/frankban/quicktest |
| github.com/fsnotify/fsnotify | v1.4.9 | BSD-3-Clause | https://github.com/fsnotify/fsnotify |
| github.com/gabriel-vasile/mimetype | v1.4.13 | MIT | https://github.com/gabriel-vasile/mimetype |
| github.com/getkin/kin-openapi | v0.134.0 | MIT | https://github.com/getkin/kin-openapi |
| github.com/gin-contrib/cors | v1.7.6 | MIT | https://github.com/gin-contrib/cors |
| github.com/gin-contrib/gzip | v1.2.5 | MIT | https://github.com/gin-contrib/gzip |
| github.com/gin-contrib/sse | v1.1.0 | MIT | https://github.com/gin-contrib/sse |
| github.com/gin-gonic/gin | v1.12.0 | MIT | https://github.com/gin-gonic/gin |
| github.com/glebarez/go-sqlite | v1.22.0 | BSD | https://github.com/glebarez/go-sqlite |
| github.com/glebarez/sqlite | v1.11.0 | MIT | https://github.com/glebarez/sqlite |
| github.com/go-jose/go-jose/v4 | v4.1.3 | Apache-2.0 | https://github.com/go-jose/go-jose/v4 |
| github.com/go-logr/logr | v1.4.3 | Apache-2.0 | https://github.com/go-logr/logr |
| github.com/go-logr/stdr | v1.2.2 | Apache-2.0 | https://github.com/go-logr/stdr |
| github.com/go-ole/go-ole | v1.2.6 | MIT | https://github.com/go-ole/go-ole |
| github.com/go-openapi/jsonpointer | v0.22.5 | Apache-2.0 | https://github.com/go-openapi/jsonpointer |
| github.com/go-openapi/jsonreference | v0.21.5 | Apache-2.0 | https://github.com/go-openapi/jsonreference |
| github.com/go-openapi/spec | v0.22.4 | Apache-2.0 | https://github.com/go-openapi/spec |
| github.com/go-openapi/swag | v0.23.0 | Apache-2.0 | https://github.com/go-openapi/swag |
| github.com/go-openapi/swag/conv | v0.25.5 | Apache-2.0 | https://github.com/go-openapi/swag/conv |
| github.com/go-openapi/swag/jsonname | v0.25.5 | Apache-2.0 | https://github.com/go-openapi/swag/jsonname |
| github.com/go-openapi/swag/jsonutils | v0.25.5 | Apache-2.0 | https://github.com/go-openapi/swag/jsonutils |
| github.com/go-openapi/swag/jsonutils/fixtures_test | v0.25.5 | Apache-2.0 | https://github.com/go-openapi/swag/jsonutils/fixtures_test |
| github.com/go-openapi/swag/loading | v0.25.5 | Apache-2.0 | https://github.com/go-openapi/swag/loading |
| github.com/go-openapi/swag/stringutils | v0.25.5 | Apache-2.0 | https://github.com/go-openapi/swag/stringutils |
| github.com/go-openapi/swag/typeutils | v0.25.5 | Apache-2.0 | https://github.com/go-openapi/swag/typeutils |
| github.com/go-openapi/swag/yamlutils | v0.25.5 | Apache-2.0 | https://github.com/go-openapi/swag/yamlutils |
| github.com/go-openapi/testify/enable/yaml/v2 | v2.4.0 | Apache-2.0 | https://github.com/go-openapi/testify/enable/yaml/v2 |
| github.com/go-openapi/testify/v2 | v2.4.0 | Apache-2.0 | https://github.com/go-openapi/testify/v2 |
| github.com/go-playground/assert/v2 | v2.2.0 | MIT | https://github.com/go-playground/assert/v2 |
| github.com/go-playground/locales | v0.14.1 | MIT | https://github.com/go-playground/locales |
| github.com/go-playground/universal-translator | v0.18.1 | MIT | https://github.com/go-playground/universal-translator |
| github.com/go-playground/validator/v10 | v10.30.1 | MIT | https://github.com/go-playground/validator/v10 |
| github.com/go-shiori/dom | v0.0.0-20230515143342-73569d674e1c | MIT | https://github.com/go-shiori/dom |
| github.com/go-shiori/go-readability | v0.0.0-20251205110129-5db1dc9836f0 | MIT | https://github.com/go-shiori/go-readability |
| github.com/go-sql-driver/mysql | v1.9.3 | MPL-2.0 | https://github.com/go-sql-driver/mysql |
| github.com/go-test/deep | v1.1.1 | MIT | https://github.com/go-test/deep |
| github.com/go-viper/mapstructure/v2 | v2.5.0 | MIT | https://github.com/go-viper/mapstructure/v2 |
| github.com/goccy/go-json | v0.10.6 | MIT | https://github.com/goccy/go-json |
| github.com/goccy/go-yaml | v1.19.2 | MIT | https://github.com/goccy/go-yaml |
| github.com/gogo/protobuf | v1.3.2 | BSD-3-Clause | https://github.com/gogo/protobuf |
| github.com/gogs/chardet | v0.0.0-20211120154057-b7413eaefb8f | MIT | https://github.com/gogs/chardet |
| github.com/golang-jwt/jwt/v5 | v5.3.1 | MIT | https://github.com/golang-jwt/jwt/v5 |
| github.com/golang/glog | v1.2.5 | Apache-2.0 | https://github.com/golang/glog |
| github.com/golang/groupcache | v0.0.0-20210331224755-41bb18bfe9da | Apache-2.0 | https://github.com/golang/groupcache |
| github.com/golang/protobuf | v1.5.4 | BSD-3-Clause | https://github.com/golang/protobuf |
| github.com/google/go-cmp | v0.7.0 | BSD-3-Clause | https://github.com/google/go-cmp |
| github.com/google/go-pkcs11 | v0.3.0 | Apache-2.0 | https://github.com/google/go-pkcs11 |
| github.com/google/gofuzz | v1.0.0 | Apache-2.0 | https://github.com/google/gofuzz |
| github.com/google/jsonschema-go | v0.4.2 | MIT | https://github.com/google/jsonschema-go |
| github.com/google/pprof | v0.0.0-20250317173921-a4b03ec1a45e | Apache-2.0 | https://github.com/google/pprof |
| github.com/google/s2a-go | v0.1.9 | Apache-2.0 | https://github.com/google/s2a-go |
| github.com/google/uuid | v1.6.0 | BSD-3-Clause | https://github.com/google/uuid |
| github.com/googleapis/enterprise-certificate-proxy | v0.3.14 | Apache-2.0 | https://github.com/googleapis/enterprise-certificate-proxy |
| github.com/googleapis/gax-go/v2 | v2.20.0 | BSD-3-Clause | https://github.com/googleapis/gax-go/v2 |
| github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp | v1.30.0 | Apache-2.0 | https://github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp |
| github.com/gopherjs/gopherjs | v1.17.2 | BSD | https://github.com/gopherjs/gopherjs |
| github.com/gorilla/mux | v1.8.0 | BSD-3-Clause | https://github.com/gorilla/mux |
| github.com/gorilla/websocket | v1.5.3 | BSD | https://github.com/gorilla/websocket |
| github.com/grpc-ecosystem/grpc-gateway/v2 | v2.28.0 | BSD-3-Clause | https://github.com/grpc-ecosystem/grpc-gateway/v2 |
| github.com/hashicorp/golang-lru/v2 | v2.0.7 | MPL-2.0 | https://github.com/hashicorp/golang-lru/v2 |
| github.com/inconshreveable/mousetrap | v1.1.0 | Apache-2.0 | https://github.com/inconshreveable/mousetrap |
| github.com/jackc/pgpassfile | v1.0.0 | MIT | https://github.com/jackc/pgpassfile |
| github.com/jackc/pgservicefile | v0.0.0-20240606120523-5a60cdf6a761 | MIT | https://github.com/jackc/pgservicefile |
| github.com/jackc/pgx/v5 | v5.9.1 | MIT | https://github.com/jackc/pgx/v5 |
| github.com/jackc/puddle/v2 | v2.2.2 | MIT | https://github.com/jackc/puddle/v2 |
| github.com/jinzhu/inflection | v1.0.0 | MIT | https://github.com/jinzhu/inflection |
| github.com/jinzhu/now | v1.1.5 | MIT | https://github.com/jinzhu/now |
| github.com/JohannesKaufmann/html-to-markdown | v1.6.0 | MIT | https://github.com/JohannesKaufmann/html-to-markdown |
| github.com/joho/godotenv | v1.5.1 | UNKNOWN | https://github.com/joho/godotenv |
| github.com/jordanlewis/gcassert | v0.0.0-20250430164644-389ef753e22e | MIT | https://github.com/jordanlewis/gcassert |
| github.com/josharian/intern | v1.0.0 | MIT | https://github.com/josharian/intern |
| github.com/json-iterator/go | v1.1.12 | MIT | https://github.com/json-iterator/go |
| github.com/jtolds/gls | v4.20.0+incompatible | MIT | https://github.com/jtolds/gls |
| github.com/kballard/go-shellquote | v0.0.0-20180428030007-95032a82bc51 | MIT | https://github.com/kballard/go-shellquote |
| github.com/kisielk/errcheck | v1.5.0 | MIT | https://github.com/kisielk/errcheck |
| github.com/kisielk/gotool | v1.0.0 | MIT | https://github.com/kisielk/gotool |
| github.com/klauspost/compress | v1.18.0 | Apache-2.0 | https://github.com/klauspost/compress |
| github.com/klauspost/cpuid/v2 | v2.3.0 | MIT | https://github.com/klauspost/cpuid/v2 |
| github.com/kr/pretty | v0.3.1 | MIT | https://github.com/kr/pretty |
| github.com/kr/pty | v1.1.1 | MIT | https://github.com/kr/pty |
| github.com/kr/text | v0.2.0 | MIT | https://github.com/kr/text |
| github.com/KyleBanks/depth | v1.2.1 | MIT | https://github.com/KyleBanks/depth |
| github.com/larksuite/oapi-sdk-go/v3 | v3.5.3 | MIT | https://github.com/larksuite/oapi-sdk-go/v3 |
| github.com/leodido/go-urn | v1.4.0 | MIT | https://github.com/leodido/go-urn |
| github.com/lufia/plan9stats | v0.0.0-20211012122336-39d0f177ccd0 | BSD-3-Clause | https://github.com/lufia/plan9stats |
| github.com/magiconair/properties | v1.8.10 | BSD | https://github.com/magiconair/properties |
| github.com/mailru/easyjson | v0.9.2 | MIT | https://github.com/mailru/easyjson |
| github.com/mark3labs/mcp-go | v0.46.0 | MIT | https://github.com/mark3labs/mcp-go |
| github.com/maruel/natural | v1.3.0 | Apache-2.0 | https://github.com/maruel/natural |
| github.com/mattn/go-isatty | v0.0.20 | MIT | https://github.com/mattn/go-isatty |
| github.com/mattn/go-runewidth | v0.0.10 | MIT | https://github.com/mattn/go-runewidth |
| github.com/mattn/go-sqlite3 | v1.14.22 | MIT | https://github.com/mattn/go-sqlite3 |
| github.com/Microsoft/go-winio | v0.6.2 | MIT | https://github.com/Microsoft/go-winio |
| github.com/moby/docker-image-spec | v1.3.1 | Apache-2.0 | https://github.com/moby/docker-image-spec |
| github.com/moby/go-archive | v0.1.0 | Apache-2.0 | https://github.com/moby/go-archive |
| github.com/moby/patternmatcher | v0.6.0 | Apache-2.0 | https://github.com/moby/patternmatcher |
| github.com/moby/sys/atomicwriter | v0.1.0 | Apache-2.0 | https://github.com/moby/sys/atomicwriter |
| github.com/moby/sys/mount | v0.3.4 | UNKNOWN | https://github.com/moby/sys/mount |
| github.com/moby/sys/mountinfo | v0.7.2 | UNKNOWN | https://github.com/moby/sys/mountinfo |
| github.com/moby/sys/reexec | v0.1.0 | UNKNOWN | https://github.com/moby/sys/reexec |
| github.com/moby/sys/sequential | v0.6.0 | Apache-2.0 | https://github.com/moby/sys/sequential |
| github.com/moby/sys/user | v0.4.0 | Apache-2.0 | https://github.com/moby/sys/user |
| github.com/moby/sys/userns | v0.1.0 | Apache-2.0 | https://github.com/moby/sys/userns |
| github.com/moby/term | v0.5.0 | Apache-2.0 | https://github.com/moby/term |
| github.com/modern-go/concurrent | v0.0.0-20180306012644-bacd9c7ef1dd | Apache-2.0 | https://github.com/modern-go/concurrent |
| github.com/modern-go/reflect2 | v1.0.2 | Apache-2.0 | https://github.com/modern-go/reflect2 |
| github.com/mohae/deepcopy | v0.0.0-20170929034955-c48cc78d4826 | MIT | https://github.com/mohae/deepcopy |
| github.com/morikuni/aec | v1.0.0 | MIT | https://github.com/morikuni/aec |
| github.com/ncruces/go-strftime | v1.0.0 | MIT | https://github.com/ncruces/go-strftime |
| github.com/neelance/astrewrite | v0.0.0-20160511093645-99348263ae86 | BSD | https://github.com/neelance/astrewrite |
| github.com/neelance/sourcemap | v0.0.0-20200213170602-2833bce08e4c | BSD | https://github.com/neelance/sourcemap |
| github.com/oasdiff/yaml | v0.0.1 | MIT | https://github.com/oasdiff/yaml |
| github.com/oasdiff/yaml3 | v0.0.1 | Apache-2.0 | https://github.com/oasdiff/yaml3 |
| github.com/opencontainers/go-digest | v1.0.0 | Apache-2.0 | https://github.com/opencontainers/go-digest |
| github.com/opencontainers/image-spec | v1.1.1 | Apache-2.0 | https://github.com/opencontainers/image-spec |
| github.com/patrickmn/go-cache | v2.1.0+incompatible | MIT | https://github.com/patrickmn/go-cache |
| github.com/pelletier/go-toml/v2 | v2.3.0 | MIT | https://github.com/pelletier/go-toml/v2 |
| github.com/perimeterx/marshmallow | v1.1.5 | MIT | https://github.com/perimeterx/marshmallow |
| github.com/pkg/errors | v0.9.1 | BSD | https://github.com/pkg/errors |
| github.com/planetscale/vtprotobuf | v0.6.1-0.20240319094008-0393e58bdf10 | BSD-3-Clause | https://github.com/planetscale/vtprotobuf |
| github.com/pmezard/go-difflib | v1.0.0 | BSD | https://github.com/pmezard/go-difflib |
| github.com/power-devops/perfstat | v0.0.0-20210106213030-5aafc221ea8c | MIT | https://github.com/power-devops/perfstat |
| github.com/PuerkitoBio/goquery | v1.12.0 | BSD-3-Clause | https://github.com/PuerkitoBio/goquery |
| github.com/PuerkitoBio/purell | v1.1.1 | BSD-3-Clause | https://github.com/PuerkitoBio/purell |
| github.com/PuerkitoBio/urlesc | v0.0.0-20170810143723-de5bf2ad4578 | BSD-3-Clause | https://github.com/PuerkitoBio/urlesc |
| github.com/quic-go/qpack | v0.6.0 | MIT | https://github.com/quic-go/qpack |
| github.com/quic-go/quic-go | v0.59.0 | MIT | https://github.com/quic-go/quic-go |
| github.com/redis/go-redis/v9 | v9.18.0 | BSD | https://github.com/redis/go-redis/v9 |
| github.com/remyoudompheng/bigfft | v0.0.0-20230129092748-24d4a6f8daec | BSD-3-Clause | https://github.com/remyoudompheng/bigfft |
| github.com/richardlehane/mscfb | v1.0.6 | Apache-2.0 | https://github.com/richardlehane/mscfb |
| github.com/richardlehane/msoleps | v1.0.6 | Apache-2.0 | https://github.com/richardlehane/msoleps |
| github.com/rivo/uniseg | v0.1.0 | MIT | https://github.com/rivo/uniseg |
| github.com/rogpeppe/go-internal | v1.14.1 | BSD-3-Clause | https://github.com/rogpeppe/go-internal |
| github.com/russross/blackfriday | v1.6.0 | UNKNOWN | https://github.com/russross/blackfriday |
| github.com/russross/blackfriday/v2 | v2.0.1 | BSD | https://github.com/russross/blackfriday/v2 |
| github.com/santhosh-tekuri/jsonschema/v5 | v5.3.1 | UNKNOWN | https://github.com/santhosh-tekuri/jsonschema/v5 |
| github.com/scylladb/termtables | v0.0.0-20191203121021-c4c0b6d42ff4 | Apache-2.0 | https://github.com/scylladb/termtables |
| github.com/sebdah/goldie/v2 | v2.5.3 | MIT | https://github.com/sebdah/goldie/v2 |
| github.com/sergi/go-diff | v1.3.1 | MIT | https://github.com/sergi/go-diff |
| github.com/shirou/gopsutil/v4 | v4.25.6 | BSD-3-Clause | https://github.com/shirou/gopsutil/v4 |
| github.com/shopspring/decimal | v1.4.0 | MIT | https://github.com/shopspring/decimal |
| github.com/shurcooL/go | v0.0.0-20200502201357-93f07166e636 | MIT | https://github.com/shurcooL/go |
| github.com/shurcooL/httpfs | v0.0.0-20190707220628-8d4bc4ba7749 | MIT | https://github.com/shurcooL/httpfs |
| github.com/shurcooL/sanitized_anchor_name | v1.0.0 | MIT | https://github.com/shurcooL/sanitized_anchor_name |
| github.com/shurcooL/vfsgen | v0.0.0-20200824052919-0d455de96546 | MIT | https://github.com/shurcooL/vfsgen |
| github.com/sirupsen/logrus | v1.9.4 | MIT | https://github.com/sirupsen/logrus |
| github.com/smarty/assertions | v1.15.0 | MIT | https://github.com/smarty/assertions |
| github.com/smartystreets/goconvey | v1.8.1 | MIT | https://github.com/smartystreets/goconvey |
| github.com/spf13/cast | v1.10.0 | MIT | https://github.com/spf13/cast |
| github.com/spf13/cobra | v1.8.1 | Apache-2.0 | https://github.com/spf13/cobra |
| github.com/spf13/pflag | v1.0.6 | BSD-3-Clause | https://github.com/spf13/pflag |
| github.com/spiffe/go-spiffe/v2 | v2.6.0 | Apache-2.0 | https://github.com/spiffe/go-spiffe/v2 |
| github.com/srwiley/oksvg | v0.0.0-20221011165216-be6e8873101c | BSD-3-Clause | https://github.com/srwiley/oksvg |
| github.com/srwiley/rasterx | v0.0.0-20220730225603-2ab79fcdd4ef | BSD-3-Clause | https://github.com/srwiley/rasterx |
| github.com/stretchr/objx | v0.5.2 | MIT | https://github.com/stretchr/objx |
| github.com/stretchr/testify | v1.11.1 | MIT | https://github.com/stretchr/testify |
| github.com/swaggo/files | v1.0.1 | MIT | https://github.com/swaggo/files |
| github.com/swaggo/gin-swagger | v1.6.1 | MIT | https://github.com/swaggo/gin-swagger |
| github.com/swaggo/swag | v1.16.6 | MIT | https://github.com/swaggo/swag |
| github.com/temoto/robotstxt | v1.1.2 | MIT | https://github.com/temoto/robotstxt |
| github.com/testcontainers/testcontainers-go | v0.39.0 | MIT | https://github.com/testcontainers/testcontainers-go |
| github.com/tiendc/go-deepcopy | v1.7.2 | MIT | https://github.com/tiendc/go-deepcopy |
| github.com/tiktoken-go/tokenizer | v0.7.0 | MIT | https://github.com/tiktoken-go/tokenizer |
| github.com/tklauser/go-sysconf | v0.3.12 | BSD-3-Clause | https://github.com/tklauser/go-sysconf |
| github.com/tklauser/numcpus | v0.6.1 | Apache-2.0 | https://github.com/tklauser/numcpus |
| github.com/twitchyliquid64/golang-asm | v0.15.1 | BSD-3-Clause | https://github.com/twitchyliquid64/golang-asm |
| github.com/ugorji/go/codec | v1.3.1 | MIT | https://github.com/ugorji/go/codec |
| github.com/urfave/cli/v2 | v2.3.0 | MIT | https://github.com/urfave/cli/v2 |
| github.com/woodsbury/decimal128 | v1.4.0 | UNKNOWN | https://github.com/woodsbury/decimal128 |
| github.com/xdg-go/pbkdf2 | v1.0.0 | UNKNOWN | https://github.com/xdg-go/pbkdf2 |
| github.com/xdg-go/scram | v1.2.0 | UNKNOWN | https://github.com/xdg-go/scram |
| github.com/xdg-go/stringprep | v1.0.4 | UNKNOWN | https://github.com/xdg-go/stringprep |
| github.com/xuri/efp | v0.0.1 | BSD-3-Clause | https://github.com/xuri/efp |
| github.com/xuri/excelize/v2 | v2.10.1 | BSD-3-Clause | https://github.com/xuri/excelize/v2 |
| github.com/xuri/nfp | v0.0.2-0.20250530014748-2ddeb826f9a9 | BSD-3-Clause | https://github.com/xuri/nfp |
| github.com/yosida95/uritemplate/v3 | v3.0.2 | BSD-3-Clause | https://github.com/yosida95/uritemplate/v3 |
| github.com/youmark/pkcs8 | v0.0.0-20240726163527-a2c0da244d78 | UNKNOWN | https://github.com/youmark/pkcs8 |
| github.com/yuin/goldmark | v1.7.1 | MIT | https://github.com/yuin/goldmark |
| github.com/yusufpapurcu/wmi | v1.2.4 | MIT | https://github.com/yusufpapurcu/wmi |
| github.com/zeebo/xxh3 | v1.0.2 | BSD | https://github.com/zeebo/xxh3 |
| go.mongodb.org/mongo-driver/v2 | v2.5.0 | Apache-2.0 |  |
| go.opencensus.io | v0.24.0 | Apache-2.0 |  |
| go.opentelemetry.io/auto/sdk | v1.2.1 | Apache-2.0 | https://go.opentelemetry.io/auto/sdk |
| go.opentelemetry.io/contrib/detectors/gcp | v1.39.0 | Apache-2.0 | https://go.opentelemetry.io/contrib/detectors/gcp |
| go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc | v0.67.0 | Apache-2.0 | https://go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc |
| go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp | v0.67.0 | Apache-2.0 | https://go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp |
| go.opentelemetry.io/otel | v1.42.0 | Apache-2.0 | https://go.opentelemetry.io/otel |
| go.opentelemetry.io/otel/exporters/otlp/otlptrace | v1.42.0 | Apache-2.0 | https://go.opentelemetry.io/otel/exporters/otlp/otlptrace |
| go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp | v1.19.0 | Apache-2.0 | https://go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp |
| go.opentelemetry.io/otel/metric | v1.42.0 | Apache-2.0 | https://go.opentelemetry.io/otel/metric |
| go.opentelemetry.io/otel/sdk | v1.42.0 | Apache-2.0 | https://go.opentelemetry.io/otel/sdk |
| go.opentelemetry.io/otel/sdk/metric | v1.42.0 | Apache-2.0 | https://go.opentelemetry.io/otel/sdk/metric |
| go.opentelemetry.io/otel/trace | v1.42.0 | Apache-2.0 | https://go.opentelemetry.io/otel/trace |
| go.opentelemetry.io/proto/otlp | v1.10.0 | Apache-2.0 | https://go.opentelemetry.io/proto/otlp |
| go.uber.org/atomic | v1.11.0 | MIT | https://go.uber.org/atomic |
| go.uber.org/mock | v0.6.0 | Apache-2.0 | https://go.uber.org/mock |
| go.yaml.in/yaml/v3 | v3.0.4 | Apache-2.0 |  |
| golang.org/x/arch | v0.25.0 | BSD-3-Clause | https://golang.org/x/arch |
| golang.org/x/crypto | v0.49.0 | BSD-3-Clause | https://golang.org/x/crypto |
| golang.org/x/exp | v0.0.0-20251023183803-a4bb9ffd2546 | UNKNOWN | https://golang.org/x/exp |
| golang.org/x/image | v0.38.0 | BSD-3-Clause | https://golang.org/x/image |
| golang.org/x/mod | v0.34.0 | BSD-3-Clause | https://golang.org/x/mod |
| golang.org/x/net | v0.52.0 | BSD-3-Clause | https://golang.org/x/net |
| golang.org/x/oauth2 | v0.36.0 | BSD-3-Clause | https://golang.org/x/oauth2 |
| golang.org/x/sync | v0.20.0 | BSD-3-Clause | https://golang.org/x/sync |
| golang.org/x/sys | v0.42.0 | BSD-3-Clause | https://golang.org/x/sys |
| golang.org/x/telemetry | v0.0.0-20260311193753-579e4da9a98c | UNKNOWN | https://golang.org/x/telemetry |
| golang.org/x/term | v0.41.0 | BSD-3-Clause | https://golang.org/x/term |
| golang.org/x/text | v0.35.0 | BSD-3-Clause | https://golang.org/x/text |
| golang.org/x/time | v0.15.0 | BSD-3-Clause | https://golang.org/x/time |
| golang.org/x/tools | v0.43.0 | BSD-3-Clause | https://golang.org/x/tools |
| golang.org/x/xerrors | v0.0.0-20231012003039-104605ab7028 | BSD-3-Clause | https://golang.org/x/xerrors |
| gonum.org/v1/gonum | v0.16.0 | BSD-3-Clause |  |
| google.golang.org/api | v0.273.0 | BSD-3-Clause | https://google.golang.org/api |
| google.golang.org/appengine | v1.6.8 | Apache-2.0 | https://google.golang.org/appengine |
| google.golang.org/genproto | v0.0.0-20260319201613-d00831a3d3e7 | Apache-2.0 | https://google.golang.org/genproto |
| google.golang.org/genproto/googleapis/api | v0.0.0-20260319201613-d00831a3d3e7 | Apache-2.0 | https://google.golang.org/genproto/googleapis/api |
| google.golang.org/genproto/googleapis/bytestream | v0.0.0-20260319201613-d00831a3d3e7 | UNKNOWN | https://google.golang.org/genproto/googleapis/bytestream |
| google.golang.org/genproto/googleapis/rpc | v0.0.0-20260319201613-d00831a3d3e7 | Apache-2.0 | https://google.golang.org/genproto/googleapis/rpc |
| google.golang.org/grpc | v1.79.3 | Apache-2.0 | https://google.golang.org/grpc |
| google.golang.org/protobuf | v1.36.11 | BSD-3-Clause | https://google.golang.org/protobuf |
| gopkg.in/check.v1 | v1.0.0-20201130134442-10cb98267c6c | BSD |  |
| gopkg.in/yaml.v2 | v2.4.0 | Apache-2.0 |  |
| gopkg.in/yaml.v3 | v3.0.1 | Apache-2.0 |  |
| gorm.io/driver/mysql | v1.6.0 | MIT |  |
| gorm.io/driver/postgres | v1.6.0 | MIT |  |
| gorm.io/driver/sqlite | v1.6.0 | MIT |  |
| gorm.io/gorm | v1.31.1 | MIT |  |
| gotest.tools/v3 | v3.5.2 | Apache-2.0 |  |
| lukechampine.com/uint128 | v1.3.0 | MIT |  |
| modernc.org/cc/v3 | v3.41.0 | BSD-3-Clause |  |
| modernc.org/cc/v4 | v4.27.1 | BSD-3-Clause |  |
| modernc.org/ccgo/v3 | v3.16.15 | BSD-3-Clause |  |
| modernc.org/ccgo/v4 | v4.32.0 | BSD-3-Clause |  |
| modernc.org/fileutil | v1.4.0 | BSD-3-Clause |  |
| modernc.org/gc/v2 | v2.6.5 | BSD-3-Clause |  |
| modernc.org/gc/v3 | v3.1.2 | BSD-3-Clause |  |
| modernc.org/goabi0 | v0.2.0 | BSD |  |
| modernc.org/libc | v1.70.0 | BSD-3-Clause |  |
| modernc.org/mathutil | v1.7.1 | BSD-3-Clause |  |
| modernc.org/memory | v1.11.0 | BSD-3-Clause |  |
| modernc.org/opt | v0.1.4 | BSD-3-Clause |  |
| modernc.org/sortutil | v1.2.1 | BSD-3-Clause |  |
| modernc.org/sqlite | v1.47.0 | BSD |  |
| modernc.org/strutil | v1.2.1 | BSD-3-Clause |  |
| modernc.org/token | v1.1.0 | BSD-3-Clause |  |
| rsc.io/pdf | v0.1.1 | BSD-3-Clause |  |
| sigs.k8s.io/yaml | v1.3.0 | MIT |  |

## Web Production Dependencies

| Package | Version | License | Homepage |
| --- | --- | --- | --- |
| @ampproject/remapping | 2.3.0 | Apache-2.0 | https://github.com/ampproject/remapping#readme |
| @babel/runtime | 7.27.6 | MIT | https://babel.dev/docs/en/next/babel-runtime |
| @date-fns/tz | 1.4.1 | MIT | https://github.com/date-fns/tz#readme |
| @esbuild/darwin-arm64 | 0.25.5 | MIT | https://github.com/evanw/esbuild#readme |
| @floating-ui/core | 1.7.1 | MIT | https://floating-ui.com |
| @floating-ui/dom | 1.7.1 | MIT | https://floating-ui.com |
| @floating-ui/react-dom | 2.1.3 | MIT | https://floating-ui.com/docs/react-dom |
| @floating-ui/utils | 0.2.9 | MIT | https://floating-ui.com |
| @hookform/resolvers | 5.0.1 | MIT | https://react-hook-form.com |
| @isaacs/fs-minipass | 4.0.1 | ISC | https://github.com/npm/fs-minipass#readme |
| @jridgewell/gen-mapping | 0.3.8 | MIT | https://github.com/jridgewell/gen-mapping#readme |
| @jridgewell/resolve-uri | 3.1.2 | MIT | https://github.com/jridgewell/resolve-uri#readme |
| @jridgewell/set-array | 1.2.1 | MIT | https://github.com/jridgewell/set-array#readme |
| @jridgewell/sourcemap-codec | 1.5.0 | MIT | https://github.com/jridgewell/sourcemap-codec#readme |
| @jridgewell/trace-mapping | 0.3.25 | MIT | https://github.com/jridgewell/trace-mapping#readme |
| @radix-ui/number | 1.1.1 | MIT | https://radix-ui.com/primitives |
| @radix-ui/primitive | 1.1.2 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-alert-dialog | 1.1.14 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-arrow | 1.1.7 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-avatar | 1.1.10 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-collapsible | 1.1.11 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-collection | 1.1.7 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-compose-refs | 1.1.2 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-context | 1.1.2 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-dialog | 1.1.14 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-direction | 1.1.1 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-dismissable-layer | 1.1.10 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-dropdown-menu | 2.1.15 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-focus-guards | 1.1.2 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-focus-scope | 1.1.7 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-id | 1.1.1 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-label | 2.1.7 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-menu | 2.1.15 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-popover | 1.1.14 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-popper | 1.2.7 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-portal | 1.1.9 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-presence | 1.1.4 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-primitive | 2.1.3 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-roving-focus | 1.1.10 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-select | 2.2.5 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-separator | 1.1.7 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-slot | 1.2.3 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-switch | 1.2.5 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-tabs | 1.1.12 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-tooltip | 1.2.7 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-use-callback-ref | 1.1.1 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-use-controllable-state | 1.2.2 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-use-effect-event | 0.0.2 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-use-escape-keydown | 1.1.1 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-use-is-hydrated | 0.1.0 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-use-layout-effect | 1.1.1 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-use-previous | 1.1.1 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-use-rect | 1.1.1 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-use-size | 1.1.1 | MIT | https://radix-ui.com/primitives |
| @radix-ui/react-visually-hidden | 1.2.3 | MIT | https://radix-ui.com/primitives |
| @radix-ui/rect | 1.1.1 | MIT | https://radix-ui.com/primitives |
| @rollup/rollup-darwin-arm64 | 4.42.0 | MIT | https://rollupjs.org/ |
| @standard-schema/utils | 0.3.0 | MIT | https://github.com/standard-schema/standard-schema#readme |
| @tabby_ai/hijri-converter | 1.0.5 | MIT |  |
| @tailwindcss/node | 4.1.8 | MIT | https://tailwindcss.com |
| @tailwindcss/oxide | 4.1.8 | MIT | https://github.com/tailwindlabs/tailwindcss#readme |
| @tailwindcss/oxide-darwin-arm64 | 4.1.8 | MIT | https://github.com/tailwindlabs/tailwindcss#readme |
| @tailwindcss/vite | 4.1.8 | MIT | https://tailwindcss.com |
| @tanstack/query-core | 5.80.6 | MIT | https://tanstack.com/query |
| @tanstack/query-devtools | 5.80.0 | MIT | https://tanstack.com/query |
| @tanstack/react-query | 5.80.6 | MIT | https://tanstack.com/query |
| @tanstack/react-query-devtools | 5.80.6 | MIT | https://tanstack.com/query |
| @tanstack/react-table | 8.21.3 | MIT | https://tanstack.com/table |
| @tanstack/table-core | 8.21.3 | MIT | https://tanstack.com/table |
| @types/debug | 4.1.12 | MIT | https://github.com/DefinitelyTyped/DefinitelyTyped/tree/master/types/debug |
| @types/estree | 1.0.7, 1.0.8 | MIT | https://github.com/DefinitelyTyped/DefinitelyTyped/tree/master/types/estree |
| @types/estree-jsx | 1.0.5 | MIT | https://github.com/DefinitelyTyped/DefinitelyTyped/tree/master/types/estree-jsx |
| @types/hast | 2.3.10, 3.0.4 | MIT | https://github.com/DefinitelyTyped/DefinitelyTyped/tree/master/types/hast |
| @types/mdast | 4.0.4 | MIT | https://github.com/DefinitelyTyped/DefinitelyTyped/tree/master/types/mdast |
| @types/ms | 2.1.0 | MIT | https://github.com/DefinitelyTyped/DefinitelyTyped/tree/master/types/ms |
| @types/node | 22.15.30 | MIT | https://github.com/DefinitelyTyped/DefinitelyTyped/tree/master/types/node |
| @types/react | 19.1.6 | MIT | https://github.com/DefinitelyTyped/DefinitelyTyped/tree/master/types/react |
| @types/react-dom | 19.1.6 | MIT | https://github.com/DefinitelyTyped/DefinitelyTyped/tree/master/types/react-dom |
| @types/unist | 2.0.11, 3.0.3 | MIT | https://github.com/DefinitelyTyped/DefinitelyTyped/tree/master/types/unist |
| @ungap/structured-clone | 1.3.0 | ISC | https://github.com/ungap/structured-clone#readme |
| aria-hidden | 1.2.6 | MIT | https://github.com/theKashey/aria-hidden#readme |
| asap | 2.0.6 | MIT | https://github.com/kriskowal/asap#readme |
| asynckit | 0.4.0 | MIT | https://github.com/alexindigo/asynckit#readme |
| axios | 1.9.0 | MIT | https://axios-http.com |
| bail | 2.0.2 | MIT | https://github.com/wooorm/bail#readme |
| base16 | 1.0.0 | MIT | https://github.com/gaearon/base16-js |
| call-bind-apply-helpers | 1.0.2 | MIT | https://github.com/ljharb/call-bind-apply-helpers#readme |
| ccount | 2.0.1 | MIT | https://github.com/wooorm/ccount#readme |
| character-entities | 1.2.4, 2.0.2 | MIT | https://github.com/wooorm/character-entities#readme |
| character-entities-html4 | 2.1.0 | MIT | https://github.com/wooorm/character-entities-html4#readme |
| character-entities-legacy | 1.1.4, 3.0.0 | MIT | https://github.com/wooorm/character-entities-legacy#readme |
| character-reference-invalid | 1.1.4, 2.0.1 | MIT | https://github.com/wooorm/character-reference-invalid#readme |
| chownr | 3.0.0 | BlueOak-1.0.0 | https://github.com/isaacs/chownr#readme |
| class-variance-authority | 0.7.1 | Apache-2.0 | https://github.com/joe-bell/cva#readme |
| clsx | 2.1.1 | MIT | https://github.com/lukeed/clsx#readme |
| combined-stream | 1.0.8 | MIT | https://github.com/felixge/node-combined-stream |
| comma-separated-tokens | 1.0.8, 2.0.3 | MIT | https://github.com/wooorm/comma-separated-tokens#readme |
| compute-scroll-into-view | 3.1.1 | MIT | https://scroll-into-view.dev |
| cookie | 1.0.2 | MIT | https://github.com/jshttp/cookie#readme |
| cross-fetch | 3.2.0, 4.0.0 | MIT | https://github.com/lquixada/cross-fetch |
| csstype | 3.1.3 | MIT | https://github.com/frenic/csstype#readme |
| date-fns | 4.1.0 | MIT | https://github.com/date-fns/date-fns#readme |
| date-fns-jalali | 4.1.0-0 | MIT | https://github.com/date-fns-jalali/date-fns-jalali#readme |
| debug | 4.4.1 | MIT | https://github.com/debug-js/debug#readme |
| decode-named-character-reference | 1.1.0 | MIT | https://github.com/wooorm/decode-named-character-reference#readme |
| delayed-stream | 1.0.0 | MIT | https://github.com/felixge/node-delayed-stream |
| dequal | 2.0.3 | MIT | https://github.com/lukeed/dequal#readme |
| detect-libc | 2.0.4 | Apache-2.0 | https://github.com/lovell/detect-libc#readme |
| detect-node-es | 1.1.0 | MIT | https://github.com/thekashey/detect-node |
| devlop | 1.1.0 | MIT | https://github.com/wooorm/devlop#readme |
| downshift | 9.0.9 | MIT | https://downshift-js.com |
| dunder-proto | 1.0.1 | MIT | https://github.com/es-shims/dunder-proto#readme |
| echarts | 5.6.0 | Apache-2.0 | https://echarts.apache.org |
| enhanced-resolve | 5.18.1 | MIT | http://github.com/webpack/enhanced-resolve |
| es-define-property | 1.0.1 | MIT | https://github.com/ljharb/es-define-property#readme |
| es-errors | 1.3.0 | MIT | https://github.com/ljharb/es-errors#readme |
| es-object-atoms | 1.1.1 | MIT | https://github.com/ljharb/es-object-atoms#readme |
| es-set-tostringtag | 2.1.0 | MIT | https://github.com/es-shims/es-set-tostringtag#readme |
| esbuild | 0.25.5 | MIT | https://github.com/evanw/esbuild#readme |
| escape-string-regexp | 5.0.0 | MIT | https://github.com/sindresorhus/escape-string-regexp#readme |
| estree-util-is-identifier-name | 3.0.0 | MIT | https://github.com/syntax-tree/estree-util-is-identifier-name#readme |
| extend | 3.0.2 | MIT | https://github.com/justmoon/node-extend#readme |
| fault | 1.0.4 | MIT | https://github.com/wooorm/fault#readme |
| fbemitter | 3.0.0 | BSD-3-Clause | https://github.com/facebook/emitter#readme |
| fbjs | 3.0.5 | MIT | https://github.com/facebook/fbjs#readme |
| fbjs-css-vars | 1.0.2 | MIT | https://github.com/facebook/fbjs#readme |
| fdir | 6.4.5 | MIT | https://github.com/thecodrr/fdir#readme |
| flux | 4.0.4 | BSD-3-Clause | https://facebookarchive.github.io/flux/ |
| follow-redirects | 1.15.9 | MIT | https://github.com/follow-redirects/follow-redirects |
| form-data | 4.0.3 | MIT | https://github.com/form-data/form-data#readme |
| format | 0.2.2 | MIT | http://samhuri.net/proj/format |
| framer-motion | 12.16.0 | MIT | https://github.com/motiondivision/motion#readme |
| fsevents | 2.3.3 | MIT | https://github.com/fsevents/fsevents |
| function-bind | 1.1.2 | MIT | https://github.com/Raynos/function-bind |
| get-intrinsic | 1.3.0 | MIT | https://github.com/ljharb/get-intrinsic#readme |
| get-nonce | 1.0.1 | MIT | https://github.com/theKashey/get-nonce |
| get-proto | 1.0.1 | MIT | https://github.com/ljharb/get-proto#readme |
| gopd | 1.2.0 | MIT | https://github.com/ljharb/gopd#readme |
| graceful-fs | 4.2.11 | ISC | https://github.com/isaacs/node-graceful-fs#readme |
| has-symbols | 1.1.0 | MIT | https://github.com/ljharb/has-symbols#readme |
| has-tostringtag | 1.0.2 | MIT | https://github.com/inspect-js/has-tostringtag#readme |
| hasown | 2.0.2 | MIT | https://github.com/inspect-js/hasOwn#readme |
| hast-util-parse-selector | 2.2.5 | MIT | https://github.com/syntax-tree/hast-util-parse-selector#readme |
| hast-util-to-jsx-runtime | 2.3.6 | MIT | https://github.com/syntax-tree/hast-util-to-jsx-runtime#readme |
| hast-util-whitespace | 3.0.0 | MIT | https://github.com/syntax-tree/hast-util-whitespace#readme |
| hastscript | 6.0.0 | MIT | https://github.com/syntax-tree/hastscript#readme |
| highlight.js | 10.7.3 | BSD-3-Clause | https://highlightjs.org/ |
| highlightjs-vue | 1.0.0 | CC0-1.0 | https://github.com/highlightjs/highlightjs-vue#readme |
| html-parse-stringify | 3.0.1 | MIT | https://github.com/henrikjoreteg/html-parse-stringify |
| html-url-attributes | 3.0.1 | MIT | https://github.com/rehypejs/rehype-minify/tree/main#readme |
| i18next | 25.2.1 | MIT | https://www.i18next.com |
| i18next-browser-languagedetector | 8.1.0 | MIT | https://github.com/i18next/i18next-browser-languageDetector |
| i18next-http-backend | 3.0.2 | MIT | https://github.com/i18next/i18next-http-backend |
| inline-style-parser | 0.2.4 | MIT | https://github.com/remarkablemark/inline-style-parser#readme |
| is-alphabetical | 1.0.4, 2.0.1 | MIT | https://github.com/wooorm/is-alphabetical#readme |
| is-alphanumerical | 1.0.4, 2.0.1 | MIT | https://github.com/wooorm/is-alphanumerical#readme |
| is-decimal | 1.0.4, 2.0.1 | MIT | https://github.com/wooorm/is-decimal#readme |
| is-hexadecimal | 1.0.4, 2.0.1 | MIT | https://github.com/wooorm/is-hexadecimal#readme |
| is-plain-obj | 4.1.0 | MIT | https://github.com/sindresorhus/is-plain-obj#readme |
| jiti | 2.4.2 | MIT | https://github.com/unjs/jiti#readme |
| js-tokens | 4.0.0 | MIT | https://github.com/lydell/js-tokens#readme |
| lightningcss | 1.30.1 | MPL-2.0 | https://github.com/parcel-bundler/lightningcss#readme |
| lightningcss-darwin-arm64 | 1.30.1 | MPL-2.0 | https://github.com/parcel-bundler/lightningcss#readme |
| lodash.curry | 4.1.1 | MIT | https://lodash.com/ |
| lodash.flow | 3.5.0 | MIT | https://lodash.com/ |
| longest-streak | 3.1.0 | MIT | https://github.com/wooorm/longest-streak#readme |
| loose-envify | 1.4.0 | MIT | https://github.com/zertosh/loose-envify |
| lowlight | 1.20.0 | MIT | https://github.com/wooorm/lowlight#readme |
| lucide-react | 0.503.0 | ISC | https://lucide.dev |
| magic-string | 0.30.17 | MIT | https://github.com/rich-harris/magic-string#readme |
| markdown-table | 3.0.4 | MIT | https://github.com/wooorm/markdown-table#readme |
| math-intrinsics | 1.1.0 | MIT | https://github.com/es-shims/math-intrinsics#readme |
| mdast-util-find-and-replace | 3.0.2 | MIT | https://github.com/syntax-tree/mdast-util-find-and-replace#readme |
| mdast-util-from-markdown | 2.0.2 | MIT | https://github.com/syntax-tree/mdast-util-from-markdown#readme |
| mdast-util-gfm | 3.1.0 | MIT | https://github.com/syntax-tree/mdast-util-gfm#readme |
| mdast-util-gfm-autolink-literal | 2.0.1 | MIT | https://github.com/syntax-tree/mdast-util-gfm-autolink-literal#readme |
| mdast-util-gfm-footnote | 2.1.0 | MIT | https://github.com/syntax-tree/mdast-util-gfm-footnote#readme |
| mdast-util-gfm-strikethrough | 2.0.0 | MIT | https://github.com/syntax-tree/mdast-util-gfm-strikethrough#readme |
| mdast-util-gfm-table | 2.0.0 | MIT | https://github.com/syntax-tree/mdast-util-gfm-table#readme |
| mdast-util-gfm-task-list-item | 2.0.0 | MIT | https://github.com/syntax-tree/mdast-util-gfm-task-list-item#readme |
| mdast-util-mdx-expression | 2.0.1 | MIT | https://github.com/syntax-tree/mdast-util-mdx-expression#readme |
| mdast-util-mdx-jsx | 3.2.0 | MIT | https://github.com/syntax-tree/mdast-util-mdx-jsx#readme |
| mdast-util-mdxjs-esm | 2.0.1 | MIT | https://github.com/syntax-tree/mdast-util-mdxjs-esm#readme |
| mdast-util-phrasing | 4.1.0 | MIT | https://github.com/syntax-tree/mdast-util-phrasing#readme |
| mdast-util-to-hast | 13.2.0 | MIT | https://github.com/syntax-tree/mdast-util-to-hast#readme |
| mdast-util-to-markdown | 2.1.2 | MIT | https://github.com/syntax-tree/mdast-util-to-markdown#readme |
| mdast-util-to-string | 4.0.0 | MIT | https://github.com/syntax-tree/mdast-util-to-string#readme |
| micromark | 4.0.2 | MIT | https://github.com/micromark/micromark/tree/main#readme |
| micromark-core-commonmark | 2.0.3 | MIT | https://github.com/micromark/micromark/tree/main#readme |
| micromark-extension-gfm | 3.0.0 | MIT | https://github.com/micromark/micromark-extension-gfm#readme |
| micromark-extension-gfm-autolink-literal | 2.1.0 | MIT | https://github.com/micromark/micromark-extension-gfm-autolink-literal#readme |
| micromark-extension-gfm-footnote | 2.1.0 | MIT | https://github.com/micromark/micromark-extension-gfm-footnote#readme |
| micromark-extension-gfm-strikethrough | 2.1.0 | MIT | https://github.com/micromark/micromark-extension-gfm-strikethrough#readme |
| micromark-extension-gfm-table | 2.1.1 | MIT | https://github.com/micromark/micromark-extension-gfm-table#readme |
| micromark-extension-gfm-tagfilter | 2.0.0 | MIT | https://github.com/micromark/micromark-extension-gfm-tagfilter#readme |
| micromark-extension-gfm-task-list-item | 2.1.0 | MIT | https://github.com/micromark/micromark-extension-gfm-task-list-item#readme |
| micromark-factory-destination | 2.0.1 | MIT | https://github.com/micromark/micromark/tree/main#readme |
| micromark-factory-label | 2.0.1 | MIT | https://github.com/micromark/micromark/tree/main#readme |
| micromark-factory-space | 2.0.1 | MIT | https://github.com/micromark/micromark/tree/main#readme |
| micromark-factory-title | 2.0.1 | MIT | https://github.com/micromark/micromark/tree/main#readme |
| micromark-factory-whitespace | 2.0.1 | MIT | https://github.com/micromark/micromark/tree/main#readme |
| micromark-util-character | 2.1.1 | MIT | https://github.com/micromark/micromark/tree/main#readme |
| micromark-util-chunked | 2.0.1 | MIT | https://github.com/micromark/micromark/tree/main#readme |
| micromark-util-classify-character | 2.0.1 | MIT | https://github.com/micromark/micromark/tree/main#readme |
| micromark-util-combine-extensions | 2.0.1 | MIT | https://github.com/micromark/micromark/tree/main#readme |
| micromark-util-decode-numeric-character-reference | 2.0.2 | MIT | https://github.com/micromark/micromark/tree/main#readme |
| micromark-util-decode-string | 2.0.1 | MIT | https://github.com/micromark/micromark/tree/main#readme |
| micromark-util-encode | 2.0.1 | MIT | https://github.com/micromark/micromark/tree/main#readme |
| micromark-util-html-tag-name | 2.0.1 | MIT | https://github.com/micromark/micromark/tree/main#readme |
| micromark-util-normalize-identifier | 2.0.1 | MIT | https://github.com/micromark/micromark/tree/main#readme |
| micromark-util-resolve-all | 2.0.1 | MIT | https://github.com/micromark/micromark/tree/main#readme |
| micromark-util-sanitize-uri | 2.0.1 | MIT | https://github.com/micromark/micromark/tree/main#readme |
| micromark-util-subtokenize | 2.1.0 | MIT | https://github.com/micromark/micromark/tree/main#readme |
| micromark-util-symbol | 2.0.1 | MIT | https://github.com/micromark/micromark/tree/main#readme |
| micromark-util-types | 2.0.2 | MIT | https://github.com/micromark/micromark/tree/main#readme |
| mime-db | 1.52.0 | MIT | https://github.com/jshttp/mime-db#readme |
| mime-types | 2.1.35 | MIT | https://github.com/jshttp/mime-types#readme |
| minipass | 7.1.2 | ISC | https://github.com/isaacs/minipass#readme |
| minizlib | 3.0.2 | MIT | https://github.com/isaacs/minizlib#readme |
| mkdirp | 3.0.1 | MIT | https://github.com/isaacs/node-mkdirp#readme |
| motion | 12.16.0 | MIT | https://github.com/motiondivision/motion#readme |
| motion-dom | 12.16.0 | MIT | https://github.com/motiondivision/motion#readme |
| motion-utils | 12.12.1 | MIT | https://github.com/motiondivision/motion#readme |
| ms | 2.1.3 | MIT | https://github.com/vercel/ms#readme |
| nanoid | 3.3.11 | MIT | https://github.com/ai/nanoid#readme |
| next-themes | 0.4.6 | MIT | https://github.com/pacocoursey/next-themes#readme |
| node-fetch | 2.7.0 | MIT | https://github.com/bitinn/node-fetch |
| object-assign | 4.1.1 | MIT | https://github.com/sindresorhus/object-assign#readme |
| parse-entities | 2.0.0, 4.0.2 | MIT | https://github.com/wooorm/parse-entities#readme |
| picocolors | 1.1.1 | ISC | https://github.com/alexeyraspopov/picocolors#readme |
| picomatch | 4.0.2 | MIT | https://github.com/micromatch/picomatch |
| postcss | 8.5.4 | MIT | https://postcss.org/ |
| prismjs | 1.27.0, 1.30.0 | MIT | https://github.com/PrismJS/prism#readme |
| promise | 7.3.1 | MIT | https://github.com/then/promise#readme |
| prop-types | 15.8.1 | MIT | https://facebook.github.io/react/ |
| property-information | 5.6.0, 7.1.0 | MIT | https://github.com/wooorm/property-information#readme |
| proxy-from-env | 1.1.0 | MIT | https://github.com/Rob--W/proxy-from-env#readme |
| pure-color | 1.3.0 | MIT | https://github.com/WickyNilliams/pure-color#readme |
| react | 19.1.0 | MIT | https://react.dev/ |
| react-base16-styling | 0.6.0 | MIT | https://github.com/alexkuz/react-base16-styling#readme |
| react-day-picker | 9.14.0 | MIT | https://daypicker.dev |
| react-dom | 19.1.0 | MIT | https://react.dev/ |
| react-hook-form | 7.57.0 | MIT | https://react-hook-form.com |
| react-i18next | 15.5.2 | MIT | https://github.com/i18next/react-i18next |
| react-is | 16.13.1, 18.2.0 | MIT | https://reactjs.org/ |
| react-json-view | 1.21.3 | MIT | https://github.com/mac-s-g/react-json-view |
| react-lifecycles-compat | 3.0.4 | MIT | https://github.com/reactjs/react-lifecycles-compat#readme |
| react-markdown | 10.1.0 | MIT | https://github.com/remarkjs/react-markdown#readme |
| react-remove-scroll | 2.7.1 | MIT | https://github.com/theKashey/react-remove-scroll#readme |
| react-remove-scroll-bar | 2.3.8 | MIT | https://github.com/theKashey/react-remove-scroll-bar#readme |
| react-router | 7.6.2 | MIT | https://github.com/remix-run/react-router#readme |
| react-style-singleton | 2.2.3 | MIT | https://github.com/theKashey/react-style-singleton#readme |
| react-syntax-highlighter | 15.6.1 | MIT | https://github.com/react-syntax-highlighter/react-syntax-highlighter#readme |
| react-textarea-autosize | 8.5.9 | MIT | https://github.com/Andarist/react-textarea-autosize#readme |
| refractor | 3.6.0 | MIT | https://github.com/wooorm/refractor#readme |
| remark-gfm | 4.0.1 | MIT | https://github.com/remarkjs/remark-gfm#readme |
| remark-parse | 11.0.0 | MIT | https://remark.js.org |
| remark-rehype | 11.1.2 | MIT | https://github.com/remarkjs/remark-rehype#readme |
| remark-stringify | 11.0.0 | MIT | https://remark.js.org |
| rollup | 4.42.0 | MIT | https://rollupjs.org/ |
| scheduler | 0.26.0 | MIT | https://react.dev/ |
| set-cookie-parser | 2.7.1 | MIT | https://github.com/nfriedly/set-cookie-parser |
| setimmediate | 1.0.5 | MIT | https://github.com/YuzuJS/setImmediate#readme |
| sonner | 2.0.5 | MIT | https://sonner.emilkowal.ski/ |
| source-map-js | 1.2.1 | BSD-3-Clause | https://github.com/7rulnik/source-map-js |
| space-separated-tokens | 1.1.5, 2.0.2 | MIT | https://github.com/wooorm/space-separated-tokens#readme |
| stringify-entities | 4.0.4 | MIT | https://github.com/wooorm/stringify-entities#readme |
| style-to-js | 1.1.16 | MIT | https://github.com/remarkablemark/style-to-js#readme |
| style-to-object | 1.0.8 | MIT | https://github.com/remarkablemark/style-to-object#readme |
| tailwind-merge | 3.3.0 | MIT | https://github.com/dcastil/tailwind-merge |
| tailwindcss | 4.1.8 | MIT | https://tailwindcss.com |
| tapable | 2.2.2 | MIT | https://github.com/webpack/tapable |
| tar | 7.4.3 | ISC | https://github.com/isaacs/node-tar#readme |
| tinyglobby | 0.2.14 | MIT | https://github.com/SuperchupuDev/tinyglobby#readme |
| tr46 | 0.0.3 | MIT | https://github.com/Sebmaster/tr46.js#readme |
| trim-lines | 3.0.1 | MIT | https://github.com/wooorm/trim-lines#readme |
| trough | 2.2.0 | MIT | https://github.com/wooorm/trough#readme |
| tslib | 2.3.0, 2.8.1 | 0BSD | https://www.typescriptlang.org/ |
| typescript | 5.7.3 | Apache-2.0 | https://www.typescriptlang.org/ |
| ua-parser-js | 1.0.40 | MIT | https://uaparser.dev |
| undici-types | 6.21.0 | MIT | https://undici.nodejs.org |
| unified | 11.0.5 | MIT | https://unifiedjs.com |
| unist-util-is | 6.0.0 | MIT | https://github.com/syntax-tree/unist-util-is#readme |
| unist-util-position | 5.0.0 | MIT | https://github.com/syntax-tree/unist-util-position#readme |
| unist-util-stringify-position | 4.0.0 | MIT | https://github.com/syntax-tree/unist-util-stringify-position#readme |
| unist-util-visit | 5.0.0 | MIT | https://github.com/syntax-tree/unist-util-visit#readme |
| unist-util-visit-parents | 6.0.1 | MIT | https://github.com/syntax-tree/unist-util-visit-parents#readme |
| use-callback-ref | 1.3.3 | MIT | https://github.com/theKashey/use-callback-ref#readme |
| use-composed-ref | 1.4.0 | MIT | https://github.com/Andarist/use-composed-ref#readme |
| use-isomorphic-layout-effect | 1.2.1 | MIT | https://github.com/Andarist/use-isomorphic-layout-effect#readme |
| use-latest | 1.3.0 | MIT | https://github.com/Andarist/use-latest#readme |
| use-sidecar | 1.1.3 | MIT | https://github.com/theKashey/use-sidecar |
| use-sync-external-store | 1.5.0 | MIT | https://github.com/facebook/react#readme |
| vaul | 1.1.2 | MIT | https://vaul.emilkowal.ski/ |
| vfile | 6.0.3 | MIT | https://github.com/vfile/vfile#readme |
| vfile-message | 4.0.2 | MIT | https://github.com/vfile/vfile-message#readme |
| vite | 6.3.5 | MIT | https://vite.dev |
| void-elements | 3.1.0 | MIT | https://github.com/jadejs/void-elements |
| webidl-conversions | 3.0.1 | BSD-2-Clause | https://github.com/jsdom/webidl-conversions#readme |
| whatwg-url | 5.0.0 | MIT | https://github.com/jsdom/whatwg-url#readme |
| xtend | 4.0.2 | MIT | https://github.com/Raynos/xtend |
| yallist | 5.0.0 | BlueOak-1.0.0 | https://github.com/isaacs/yallist#readme |
| zod | 3.25.56 | MIT | https://zod.dev |
| zrender | 5.6.1 | BSD-3-Clause | https://github.com/ecomfe/zrender#readme |
| zustand | 5.0.5 | MIT | https://github.com/pmndrs/zustand |
| zwitch | 2.0.4 | MIT | https://github.com/wooorm/zwitch#readme |

