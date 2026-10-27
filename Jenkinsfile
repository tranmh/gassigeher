// The safety net of gassigeher, written by aqaa from what its QA run found: unit tests with coverage and its
// floor, integration tests against the databases they need, end-to-end tests, and lint, SAST and dependency
// checks (report only: a finding marks the stage unstable until it is triaged).
// Needs the plugins Pipeline, Docker Pipeline, JUnit, Coverage, Timestamper and Workspace Cleanup;
// integration stages run on an agent labelled 'docker'.
pipeline {
  agent none
  options {
    timeout(time: 90, unit: 'MINUTES')
    timestamps()
    buildDiscarder(logRotator(numToKeepStr: '30', artifactNumToKeepStr: '10'))
    disableConcurrentBuilds(abortPrevious: true)
  }
  environment {
    CI = 'true'
  }
  stages {
    stage('Tests') {
      parallel {
        stage('Unit: go') {
          agent { docker { image 'golang:1.26' } }
          options { timeout(time: 30, unit: 'MINUTES') }
          steps {
            dir('.') {
              sh '''
                set -eu
                mkdir -p test-reports
                go mod download
                go run gotest.tools/gotestsum@v1.13.0 --format testname --junitfile test-reports/unit.xml -- -count=1 -covermode=atomic -coverpkg=./... -coverprofile=test-reports/coverage.out ./...
              '''
            }
          }
          post {
            always {
              junit allowEmptyResults: true, testResults: 'test-reports/*.xml'
              recordCoverage id: 'coverage-go', name: 'Coverage go', tools: [[parser: 'GO_COV', pattern: 'test-reports/coverage.out']], sourceDirectories: [[path: '.']], qualityGates: [[threshold: 30.0, metric: 'LINE', baseline: 'PROJECT', criticality: 'FAILURE']]
              archiveArtifacts allowEmptyArchive: true, artifacts: 'test-reports/**'
            }
            cleanup {
              cleanWs deleteDirs: true, notFailBuild: true
            }
          }
        }
        stage('Integration: go') {
          agent { label 'docker' }
          options { timeout(time: 45, unit: 'MINUTES') }
          steps {
            script {
              docker.image('postgres:16-alpine').withRun('-e POSTGRES_USER=aqaa -e POSTGRES_PASSWORD=aqaa -e POSTGRES_DB=aqaa') { postgres ->
                docker.image('postgres:16-alpine').inside("--link ${postgres.id}:postgres") {
                  sh 'until pg_isready -h postgres -U aqaa -d aqaa; do sleep 1; done'
                }
                docker.image('golang:1.26').inside("--link ${postgres.id}:postgres -e 'AQAA_PG_URL=postgres://aqaa:aqaa@postgres:5432/aqaa?sslmode=disable' -e 'AQAA_PG_CONNECTION_STRING=Host=postgres;Port=5432;Database=aqaa;Username=aqaa;Password=aqaa' -e 'DATABASE_URL=postgres://aqaa:aqaa@postgres:5432/aqaa?sslmode=disable' -e 'AQAA_PG_JDBC_URL=jdbc:postgresql://postgres:5432/aqaa?user=aqaa&password=aqaa' -e AQAA_SQLITE_PATH=/tmp/aqaa.db") {
                  dir('.') {
                    sh '''
                      set -eu
                      mkdir -p test-reports
                      go mod download
                      RUN_INTEGRATION=1 INTEGRATION=1 go run gotest.tools/gotestsum@v1.13.0 --format testname --junitfile test-reports/integration.xml -- -count=1 -tags integration -run Integration ./...
                    '''
                  }
                }
              }
            }
          }
          post {
            always {
              junit allowEmptyResults: true, testResults: 'test-reports/*.xml'
              archiveArtifacts allowEmptyArchive: true, artifacts: 'test-reports/**'
            }
            cleanup {
              cleanWs deleteDirs: true, notFailBuild: true
            }
          }
        }
        stage('Unit: node') {
          agent { docker { image 'node:22-bookworm' } }
          options { timeout(time: 30, unit: 'MINUTES') }
          steps {
            dir('.') {
              sh '''
                set -eu
                mkdir -p test-reports
                npm ci
                npm install --no-save --no-audit --no-fund jest-junit@17.0.0
                JEST_JUNIT_OUTPUT_DIR=test-reports JEST_JUNIT_OUTPUT_NAME=unit.xml npx jest --ci --coverage --coverageReporters=cobertura --coverageDirectory=test-reports/coverage --reporters=default --reporters=jest-junit --testPathIgnorePatterns integration e2e
              '''
            }
          }
          post {
            always {
              junit allowEmptyResults: true, testResults: 'test-reports/*.xml'
              recordCoverage id: 'coverage-node', name: 'Coverage node', tools: [[parser: 'COBERTURA', pattern: 'test-reports/coverage/cobertura-coverage.xml']], sourceDirectories: [[path: '.']], qualityGates: [[threshold: 98.0, metric: 'LINE', baseline: 'PROJECT', criticality: 'FAILURE']]
              archiveArtifacts allowEmptyArchive: true, artifacts: 'test-reports/**'
            }
            cleanup {
              cleanWs deleteDirs: true, notFailBuild: true
            }
          }
        }
        stage('Integration: node') {
          agent { label 'docker' }
          options { timeout(time: 45, unit: 'MINUTES') }
          steps {
            script {
              docker.image('postgres:16-alpine').withRun('-e POSTGRES_USER=aqaa -e POSTGRES_PASSWORD=aqaa -e POSTGRES_DB=aqaa') { postgres ->
                docker.image('postgres:16-alpine').inside("--link ${postgres.id}:postgres") {
                  sh 'until pg_isready -h postgres -U aqaa -d aqaa; do sleep 1; done'
                }
                docker.image('node:22-bookworm').inside("--link ${postgres.id}:postgres -e 'AQAA_PG_URL=postgres://aqaa:aqaa@postgres:5432/aqaa?sslmode=disable' -e 'AQAA_PG_CONNECTION_STRING=Host=postgres;Port=5432;Database=aqaa;Username=aqaa;Password=aqaa' -e 'DATABASE_URL=postgres://aqaa:aqaa@postgres:5432/aqaa?sslmode=disable' -e 'AQAA_PG_JDBC_URL=jdbc:postgresql://postgres:5432/aqaa?user=aqaa&password=aqaa' -e AQAA_SQLITE_PATH=/tmp/aqaa.db") {
                  dir('.') {
                    sh '''
                      set -eu
                      mkdir -p test-reports
                      npm ci
                      npm install --no-save --no-audit --no-fund jest-junit@17.0.0
                      JEST_JUNIT_OUTPUT_DIR=test-reports JEST_JUNIT_OUTPUT_NAME=integration.xml npx jest --ci --reporters=default --reporters=jest-junit --testPathPattern integration
                    '''
                  }
                }
              }
            }
          }
          post {
            always {
              junit allowEmptyResults: true, testResults: 'test-reports/*.xml'
              archiveArtifacts allowEmptyArchive: true, artifacts: 'test-reports/**'
            }
            cleanup {
              cleanWs deleteDirs: true, notFailBuild: true
            }
          }
        }
        stage('Unit: node e2e-tests') {
          agent { docker { image 'node:22-bookworm' } }
          options { timeout(time: 30, unit: 'MINUTES') }
          steps {
            dir('e2e-tests') {
              sh '''
                set -eu
                mkdir -p test-reports
                npm ci
                npm install --no-save --no-audit --no-fund "@vitest/coverage-v8@$(node -p "require('vitest/package.json').version")"
                npx vitest run --coverage --coverage.reporter=cobertura --coverage.reportsDirectory=test-reports/coverage --reporter=default --reporter=junit --outputFile.junit=test-reports/unit.xml --exclude '**/*.integration.test.*' --exclude 'e2e/**'
              '''
            }
          }
          post {
            always {
              junit allowEmptyResults: true, testResults: 'e2e-tests/test-reports/*.xml'
              recordCoverage id: 'coverage-node-e2e-tests', name: 'Coverage node e2e-tests', tools: [[parser: 'COBERTURA', pattern: 'e2e-tests/test-reports/coverage/cobertura-coverage.xml']], sourceDirectories: [[path: 'e2e-tests']]
              archiveArtifacts allowEmptyArchive: true, artifacts: 'e2e-tests/test-reports/**'
            }
            cleanup {
              cleanWs deleteDirs: true, notFailBuild: true
            }
          }
        }
        stage('End-to-end: node e2e-tests') {
          agent { docker { image 'mcr.microsoft.com/playwright:v1.56.1-noble' } }
          options { timeout(time: 45, unit: 'MINUTES') }
          steps {
            dir('e2e-tests') {
              sh '''
                set -eu
                mkdir -p test-reports
                npm ci
                npm install --no-save --no-audit --no-fund "@vitest/coverage-v8@$(node -p "require('vitest/package.json').version")"
                npx playwright install chromium
                PLAYWRIGHT_JUNIT_OUTPUT_NAME=test-reports/e2e.xml npx playwright test --reporter=junit,list
              '''
            }
          }
          post {
            always {
              junit allowEmptyResults: true, testResults: 'e2e-tests/test-reports/*.xml'
              archiveArtifacts allowEmptyArchive: true, artifacts: 'e2e-tests/test-reports/**'
            }
            cleanup {
              cleanWs deleteDirs: true, notFailBuild: true
            }
          }
        }
      }
    }
    stage('Checks') {
      parallel {
        stage('Checks: docker') {
          agent { dockerfile { filename 'docker.Dockerfile'; dir 'ci/jenkins' } }
          options { timeout(time: 30, unit: 'MINUTES') }
          steps {
            dir('.') {
              sh '''
                set -eu
                mkdir -p test-reports
              '''
              catchError(buildResult: 'SUCCESS', stageResult: 'UNSTABLE', message: 'hadolint found issues') {
                sh '''
                  set -eu
                  find . -type f \\( -name Dockerfile -o -name 'Dockerfile.*' -o -name '*.Dockerfile' -o -name Containerfile \\) -not -path './.git/*' -not -path '*/node_modules/*' -print0 | xargs -0 -r hadolint
                '''
              }
            }
          }
          post {
            always { archiveArtifacts allowEmptyArchive: true, artifacts: 'test-reports/**' }
            cleanup { cleanWs deleteDirs: true, notFailBuild: true }
          }
        }
        stage('Checks: go') {
          agent { docker { image 'golang:1.26' } }
          options { timeout(time: 30, unit: 'MINUTES') }
          steps {
            dir('.') {
              sh '''
                set -eu
                mkdir -p test-reports
                go mod download
              '''
              catchError(buildResult: 'SUCCESS', stageResult: 'UNSTABLE', message: 'go vet found issues') {
                sh '''
                  set -eu
                  go vet ./...
                '''
              }
              catchError(buildResult: 'SUCCESS', stageResult: 'UNSTABLE', message: 'golangci-lint found issues') {
                sh '''
                  set -eu
                  go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0 run --output.text.path stdout --output.sarif.path test-reports/golangci.sarif ./...
                '''
              }
              catchError(buildResult: 'SUCCESS', stageResult: 'UNSTABLE', message: 'govulncheck found issues') {
                sh '''
                  set -eu
                  go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
                '''
              }
            }
          }
          post {
            always { archiveArtifacts allowEmptyArchive: true, artifacts: 'test-reports/**' }
            cleanup { cleanWs deleteDirs: true, notFailBuild: true }
          }
        }
        stage('Checks: node') {
          agent { docker { image 'node:22-bookworm' } }
          options { timeout(time: 30, unit: 'MINUTES') }
          steps {
            dir('.') {
              sh '''
                set -eu
                mkdir -p test-reports
                npm ci
                npm install --no-save --no-audit --no-fund jest-junit@17.0.0
              '''
              catchError(buildResult: 'SUCCESS', stageResult: 'UNSTABLE', message: 'eslint found issues') {
                sh '''
                  set -eu
                  if ls eslint.config.* .eslintrc* >/dev/null 2>&1; then npx eslint .; else echo 'no ESLint configuration'; fi
                '''
              }
              catchError(buildResult: 'SUCCESS', stageResult: 'UNSTABLE', message: 'npm audit found issues') {
                sh '''
                  set -eu
                  npm audit --audit-level=high --omit=dev
                '''
              }
            }
          }
          post {
            always { archiveArtifacts allowEmptyArchive: true, artifacts: 'test-reports/**' }
            cleanup { cleanWs deleteDirs: true, notFailBuild: true }
          }
        }
        stage('Checks: node e2e-tests') {
          agent { docker { image 'node:22-bookworm' } }
          options { timeout(time: 30, unit: 'MINUTES') }
          steps {
            dir('e2e-tests') {
              sh '''
                set -eu
                mkdir -p test-reports
                npm ci
                npm install --no-save --no-audit --no-fund "@vitest/coverage-v8@$(node -p "require('vitest/package.json').version")"
              '''
              catchError(buildResult: 'SUCCESS', stageResult: 'UNSTABLE', message: 'eslint found issues') {
                sh '''
                  set -eu
                  if ls eslint.config.* .eslintrc* >/dev/null 2>&1; then npx eslint .; else echo 'no ESLint configuration'; fi
                '''
              }
              catchError(buildResult: 'SUCCESS', stageResult: 'UNSTABLE', message: 'npm audit found issues') {
                sh '''
                  set -eu
                  npm audit --audit-level=high --omit=dev
                '''
              }
            }
          }
          post {
            always { archiveArtifacts allowEmptyArchive: true, artifacts: 'e2e-tests/test-reports/**' }
            cleanup { cleanWs deleteDirs: true, notFailBuild: true }
          }
        }
      }
    }
  }
}
