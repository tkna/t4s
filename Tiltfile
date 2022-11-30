load('ext://restart_process', 'docker_build_with_restart')
load('ext://cert_manager', 'deploy_cert_manager')

deploy_cert_manager(version="v1.6.1")

DOCKERFILE = '''FROM golang:alpine
    WORKDIR /
    COPY ./bin/manager /
    CMD ["/manager"]
    '''
CONTROLLER_IMG = 'localhost:5005/t4s-controller:dev'
APP_IMG = 't4s-app:dev'

def manifests():
    return 'bin/controller-gen crd rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases;'

def generate():
    return 'bin/controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./...";'

def vetfmt():
    return 'go vet ./...; go fmt ./...'

def binary():
    return 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -o bin/manager main.go'

ignore = ['*/*/zz_generated.deepcopy.go', '**/*.test', '**/*.out']
# Generate manifests and go files
local_resource('make manifests', manifests(), deps=["api", "controllers"], ignore=ignore)
local_resource('make generate', generate(), deps=["api"], ignore=ignore)

# Deploy CRDs
local_resource('CRD', manifests() + 'kustomize build config/crd | kubectl apply -f -', deps=["api"], ignore=ignore)

# Deploy manager
watch_settings(ignore=['config/crd/bases/', 'config/rbac/role.yaml', 'config/webhook/manifests.yaml'])
k8s_yaml(kustomize('./config/dev'))

local_resource('Watch&Compile', generate() + binary(), deps=['controllers', 'main.go', 'api'], ignore=ignore)

docker_build_with_restart(CONTROLLER_IMG, '.',
    dockerfile_contents=DOCKERFILE,
    entrypoint='/manager',
    only=['./bin/manager'],
    live_update=[
        sync('./bin/manager', '/manager'),
    ]
)

# Build/push the app image and recreate the pod
build = 'docker build -t ' + APP_IMG + ' -f Dockerfile.app .;'
kindload = 'kind load docker-image ' + APP_IMG + ';'
recreate = 'pod=$(kubectl get pod -l tier=app --no-headers -o custom-columns=":metadata.name"); if [ -n "$pod" ]; then kubectl delete pod $pod; fi'
local_resource('Build App', build + kindload + recreate, deps=["app"])

# Deploy a sample YAML
DIRNAME = os.path.basename(os. getcwd())
local_resource('Sample YAML', 'kubectl apply -f ./config/samples/t4s.yaml', deps=["./config/samples/t4s.yaml"], resource_deps=[DIRNAME + "-controller-manager"])
