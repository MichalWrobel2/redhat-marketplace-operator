package ci

import (
	"strings"
	json "github.com/SchemaStore/schemastore/src/schemas/json/github"
)

workflowsDir: *"./" | string @tag(workflowsDir)

workflows: [...{file: string, schema: (json.#Workflow & {})}]
workflows: [
	{
		file:   "test.yml"
		schema: unit_test
	},
	{
		file:   "bundle.yml"
		schema: bundle
	},
]

varPresetGitTag:         "${{ needs.preset.outputs.tag }}"
varPresetVersion:        "${{ needs.preset.outputs.version }}"
varPresetHash:           "${{ needs.preset.outputs.hash}}"
varPresetDockertag:      "${{ needs.preset.outputs.dockertag}}"
varPresetQuayExpiration: "${{ needs.preset.outputs.quayExpiration}}"

unit_test: _#bashWorkflow & {
	name: "Test"
	on: {
		push: {
			branches: [
				"master",
				"release/**",
				"hotfix/**",
				"develop",
				"feature/**",
				"bugfix/**",
			]
		}
	}
	env: {
		"IMAGE_REGISTRY": "quay.io/rh-marketplace"
	}
	jobs: {
		"test-unit": _#job & {
			name:      "Test"
			"runs-on": _#linuxMachine

			steps: [
				_#checkoutCode,
				_#installGo,
				_#cacheGoModules,
				_#installKubeBuilder,
				_#step & {
					name: "Test"
					run: """
							make test
						"""
				},
			]
		}
	}
}

bundle: _#bashWorkflow & {
	name: "Deploy Bundle"
	on: ["repository_dispatch"]
	env: {
		"IMAGE_REGISTRY": "quay.io/rh-marketplace"
	}
	jobs: {
		deploy: _#job & {
			name:      "Deploy Bundle"
			"runs-on": _#linuxMachine
			if:        "contains(${{ github.event.action }}: 'bundle')"
			steps: [
				_#checkoutCode & {
          with: ref: "${{github.event.client_payload.ref}}"
        },
				_#installGo,
				_#cacheGoModules,
				_#installOperatorSDK,
				_#step & {
					name: "Build bundle"
					run: """
						VERSION=$(make current-version)-${GITHUB_SHA}
						TAG=$(make current-version)-${GITHUB_SHA}
						cd v2
						make bundle bundle-stable bundle-deploy bundle-dev-index
						"""
				},
			]
		}
		publish: _#job & {
			name:      "Publish Images"
			"runs-on": _#linuxMachine
			needs: ["deploy"]
			if: "(startsWith(github.ref,'refs/heads/release/') || startsWith(github.ref,'refs/heads/hotfix/'))"
			steps: [
				_#checkoutCode,
				_#installGo,
				_#cacheGoModules,
				_#installOperatorSDK,
				_#step & {
					name: "Mirror images"
					run:  _#retagCommand
				},
			]
		}
	}
}

_#bashWorkflow: json.#Workflow & {
	jobs: [string]: defaults: run: shell: "bash"
}

// TODO: drop when cuelang.org/issue/390 is fixed.
// Declare definitions for sub-schemas
_#on:   ((json.#Workflow & {}).on & {x:   _}).x
_#job:  ((json.#Workflow & {}).jobs & {x: _}).x
_#step: ((_#job & {steps:                 _}).steps & [_])[0]

// We need at least go1.14 for code generation
_#codeGenGo: "1.14.9"

_#linuxMachine:   "ubuntu-20.04"
_#macosMachine:   "macos-10.15"
_#windowsMachine: "windows-2019"

_#testStrategy: {
	"fail-fast": false
	matrix: {
		// Use a stable version of 1.14.x for go generate
		"go-version": ["1.13.x", _#codeGenGo, "1.15.x"]
		os: [_#linuxMachine, _#macosMachine, _#windowsMachine]
	}
}

_#cancelPreviousRun: _#step & {
	name: "Cancel Previous Run"
	uses: "styfle/cancel-workflow-action@0.4.1"
	with: "access_token": "${{ github.token }}"
}

_#installGo: _#step & {
	name: "Install Go"
	uses: "actions/setup-go@v2"
	with: "go-version": _#goVersion
}

_#checkoutCode: _#step & {
	name: "Checkout code"
	uses: "actions/checkout@v2"
}

_#cacheGoModules: _#step & {
	name: "Cache Go modules"
	uses: "actions/cache@v2"
	with: {
		path:           "~/go/pkg/mod"
		key:            "${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}"
		"restore-keys": "${{ runner.os }}-\(_#goVersion)-go-"
	}
}

_#goGenerate: _#step & {
	name: "Generate"
	run:  "go generate ./..."
	// The Go version corresponds to the precise version specified in
	// the matrix. Skip windows for now until we work out why re-gen is flaky
	if: "matrix.go-version == '\(_#codeGenGo)' && matrix.os != '\(_#windowsMachine)'"
}

_#goTest: _#step & {
	name: "Test"
	run:  "go test ./..."
}

_#goTestRace: _#step & {
	name: "Test with -race"
	run:  "go test -race ./..."
}

_#goReleaseCheck: _#step & {
	name: "gorelease check"
	run:  "go run golang.org/x/exp/cmd/gorelease"
}

_#loadGitTagPushed: _#step & {
	name: "Get if gittag is pushed"
	id:   "tag"
	run: """
		VERSION=$(make current-version)
		RESULT=$(git tag --list | grep -E "$VERSION")
		IS_TAGGED=false
		if [ "$RESULT" != "" ] ; then
		  IS_TAGGED=true
		"""
}

_#branchRefPrefix: "refs/heads/"

_#setBranchOutput: [
	_#setBranchPrefixForDev,
	_#setBranchPrefixForFix,
	_#setBranchPrefixForFeature,
]

_#setBranchPrefixForDev: (_#vars._#setBranchPrefix & {
	#args: {
		name:      "dev"
		if:        "github.event_name == 'push' && github.ref == 'refs/heads/develop'"
		tagPrefix: "dev-"
	}
}).res

_#setBranchPrefixForFix: (_#vars._#setBranchPrefix & {
	#args: {
		name:           "fix"
		if:             "github.event_name == 'push' && startsWith(github.ref,'refs/heads/bugfix/')"
		tagPrefix:      "bugfix-${NAME}-"
		quayExpiration: "1w"
	}
}).res

_#setBranchPrefixForFeature: (_#vars._#setBranchPrefix & {
	#args: {
		name:           "feature"
		if:             "github.event_name == 'push' && startsWith(github.ref,'refs/heads/feature/')"
		tagPrefix:      "feat-${NAME}-"
		quayExpiration: "1w"
	}
}).res

_#installKubeBuilder: _#step & {
	name: "Install Kubebuilder"
	run: """
		os=$(go env GOOS)
		arch=$(go env GOARCH)

		# download kubebuilder and extract it to tmp
		curl -L https://go.kubebuilder.io/dl/2.3.1/${os}/${arch} | tar -xz -C /tmp/

		# move to a long-term location and put it on your path
		# (you'll need to set the KUBEBUILDER_ASSETS env var if you put it somewhere else)
		sudo mv /tmp/kubebuilder_2.3.1_${os}_${arch} /usr/local/kubebuilder
		echo "/usr/local/kubebuilder/bin" >> $GITHUB_PATH
		"""
}

_#installOperatorSDK: _#step & {
	name: "Install operatorsdk"
	run: """
		export ARCH=$(case $(arch) in x86_64) echo -n amd64 ;; aarch64) echo -n arm64 ;; *) echo -n $(arch) ;; esac)
		export OS=$(uname | awk '{print tolower($0)}')
		export OPERATOR_SDK_DL_URL=https://github.com/operator-framework/operator-sdk/releases/latest/download
		curl -LO ${OPERATOR_SDK_DL_URL}/operator-sdk_${OS}_${ARCH}
		gpg --recv-keys 052996E2A20B5C7E
		curl -LO ${OPERATOR_SDK_DL_URL}/checksums.txt
		curl -LO ${OPERATOR_SDK_DL_URL}/checksums.txt.asc
		gpg -u "Operator SDK (release) <cncf-operator-sdk@cncf.io>" --verify checksums.txt.asc
		grep operator-sdk_${OS}_${ARCH} checksums.txt | sha256sum -c -
		chmod +x operator-sdk_${OS}_${ARCH} && sudo mv operator-sdk_${OS}_${ARCH} /usr/local/bin/operator-sdk
		"""
}

_#vars: {
	// _#setBranchPrefix will set the branch prefix vars
	_#setBranchPrefix: {
		#args: {
			name:           string
			if:             string
			tagPrefix:      string
			quayExpiration: string | *""
		}
		res: _#step & {
			name: #"Setting tags for \#(#args.name)"#
			if:   #"\#(#args["if"])"#
			run:  #"""
      echo "TAGPREFIX=\#(#args.tagPrefix)" >> $GITHUB_ENV
      if [ "\#(#args.quayExpiration)" != "" ]; then
        echo "QUAY_EXPIRATION=\#(#args.quayExpiration)" >> $GITHUB_ENV
      fi
      """#
		}
	}
}

_#turnStyleStep: _#step & {
	name: "Turnstyle"
	uses: "softprops/turnstyle@v1"
	with: "continue-after-seconds": 45
	env: "GITHUB_TOKEN":            "${{ secrets.GITHUB_TOKEN }}"
}

_#goVersion: "1.15.6"
_#pcUser:    "pcUser"

_#operator: {
	name:  "redhat-marketplace-operator"
	ospid: "scan.connect.redhat.com/ospid-c93f69b6-cb04-437b-89d6-e5220ce643cd"
	pword: "pcPassword"
}

_#metering: {
	name:  "redhat-marketplace-metric-state"
	ospid: "scan.connect.redhat.com/ospid-9b9b0dbe-7adc-448e-9385-a556714a09c4"
	pword: "pcPasswordMetricState"
}

_#reporter: {
	name:  "redhat-marketplace-reporter"
	ospid: "scan.connect.redhat.com/ospid-faa0f295-e195-4bcc-a3fc-a4b97ada317e"
	pword: "pcPasswordReporter"
}

_#authchecker: {
	name:  "redhat-marketplace-authchecker"
	ospid: "scan.connect.redhat.com/ospid-ffed416e-c18d-4b88-8660-f586a4792785"
	pword: "pcPasswordAuthCheck"
}

_#images: [
	_#operator,
	_#metering,
	_#reporter,
	_#authchecker,
]

_#registry: "quay.io/rh-marketplace"

_#manifest: {
	name:  "redhat-marketplace-operator-manifest"
	ospid: "scan.connect.redhat.com/ospid-64f06656-d9d4-43ef-a227-3b9c198800a1"
	pword: "pcPasswordOperatorManifest"
}

_#repoFromTo: [ for k, v in _#images {
	pword: "\(v.pword)"
	from:  "\(_#registry)/\(v.name):$VERSION"
	to:    "\(v.ospid)/\(v.name):$VERSION"
}]

_#skopeoCopyCommands: [ for k, v in _#repoFromTo {"skopeo copy docker://\(v.from) docker://\(v.to) --dest-creds ${{secrets.matrix['\(_#pcUser)']}}:${{secrets.matrix['(v.pword)']}}"}]
_#retagCommand: strings.Join(_#skopeoCopyCommands, "\n")

#preset: _#job & {
	name:      "Preset"
	"runs-on": _#linuxMachine
	steps:     [
			_#turnStyleStep,
			_#checkoutCode,
			_#installGo,
			_#cacheGoModules] +
		_#setBranchOutput + [
			_#step & {
				name: "Get Vars"
				id:   "vars"
				run: """
					echo "::set-output name=version::$(make current-version)"
					echo "::set-output name=tag::sha-$(git rev-parse --short HEAD)"
					echo "::set-output name=hash::$(make current-version)-$(git rev-parse --short HEAD)"
					echo "::set-output name=dockertag::${TAGPREFIX}$(make current-version)-${GITHUB_SHA::8}"
					echo "::set-output name=quayExpiration::${QUAY_EXPIRATION:-never}"
					"""
			},
		]
	outputs: {
		version:        "${{ steps.vars.outputs.version }}"
		tag:            "${{ steps.vars.outputs.tag }}"
		hash:           "${{ steps.vars.outputs.hash }}"
		dockertag:      "${{ steps.vars.outputs.dockertag }}"
		quayExpiration: "${{ steps.vars.outputs.quayExpiration }}"
	}
}