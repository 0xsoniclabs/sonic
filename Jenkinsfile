// Copyright 2026 Sonic Operations Ltd
// This file is part of the Sonic Client
//
// Sonic is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Sonic is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Sonic. If not, see <http://www.gnu.org/licenses/>.

pipeline {
    agent {
        dockerfile {
            filename 'CI/Dockerfile.jenkins'
            label 'pr'
        }
    }

    options {
        timestamps()
        timeout(time: 1, unit: 'HOURS')
        disableConcurrentBuilds(abortPrevious: true)
    }

    stages {
        stage('Build') {
            steps {
                sh 'make'
            }
        }

        stage('Check mocks are up-to-date') {
            steps {
                sh 'go generate -run \'mockgen\' ./...'
                // Detect both modifications to tracked files and any
                // newly generated untracked files.
                sh '''
                    status=$(git status --porcelain)
                    if [ -n "$status" ]; then
                        echo "Generated files are out of date:"
                        echo "$status"
                        git diff
                        exit 1
                    fi
                '''
            }
        }


        stage('Run tests') {
            steps {
                sh 'make coverage'
            }
        }

        stage('Upload test coverage') {
            environment {
                CODECOV_TOKEN = credentials('codecov-uploader-0xsoniclabs-global')
            }
            steps {
                sh("codecov upload-process -r 0xsoniclabs/sonic -f ./build/coverage.cov -t $CODECOV_TOKEN")
            }
        }
    }

    post {
        always {
            sh 'make clean'
        }
    }
}
