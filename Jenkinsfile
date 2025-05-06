pipeline {
    agent { label 'pr' }

    options {
        timestamps()
        timeout(time: 1, unit: 'HOURS')
        disableConcurrentBuilds(abortPrevious: true)
    }

    stages {
        stage('Validate commit') {
            steps {
                script {
                    def CHANGE_REPO = sh(script: 'basename -s .git `git config --get remote.origin.url`', returnStdout: true).trim()
                    build job: '/Utils/Validate-Git-Commit', parameters: [
                        string(name: 'Repo', value: "${CHANGE_REPO}"),
                        string(name: 'Branch', value: "${env.CHANGE_BRANCH}"),
                        string(name: 'Commit', value: "${GIT_COMMIT}")
                    ]
                }
            }
        }

        stage('Static analysis') {
            steps {
                sh 'make lint'
            }
        }

        stage('Build') {
            steps {
                sh 'make'
            }
        }

        stage('Run tests') {
            environment {
                CODECOV_TOKEN = credentials('codecov-uploader-0xsoniclabs-global')
            }
            steps {
                sh 'go test -coverprofile=coverage.txt --timeout 30m $(go list ./... | grep -v /tests)'
                sh 'make integration-coverage'
                sh ('codecov upload-process -r 0xsoniclabs/sonic -f ./coverage.tx -t ${CODECOV_TOKEN}')
                sh ('codecov upload-process -r 0xsoniclabs/sonic -f ./build/coverage/*/integration-cover.out -t ${CODECOV_TOKEN}')
            }
        }

        stage('Clean up') {
            steps {
                sh 'make clean'
            }
        }
    }
}
