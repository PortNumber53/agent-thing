pipeline {
    agent any

    environment {
        PROJECT_NAME = 'agent-thing'
        REMOTE_HOST = 'pinky'
        REMOTE_USER = 'grimlock'
        REMOTE_OWNER = 'grimlock'
        REMOTE_BASE_DIR = '/var/www/vhosts'
        SSH_CREDENTIALS_ID = 'pinky-ssh-key'
    }

    options {
        timestamps()
    }

    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }

        stage('Test') {
            steps {
                sh 'go test ./...'
            }
        }

        stage('Build Backend') {
            steps {
                sh '''
                    set -euo pipefail
                    mkdir -p build/bin
                    go build -o build/bin/agent-thing ./agent.go
                '''
            }
        }

        stage('Build Frontend') {
            steps {
                dir('frontend') {
                    sh 'npm ci'
                    sh 'npm run build'
                }
            }
        }

        stage('Package Release') {
            steps {
                script {
                    env.BUILD_TIMESTAMP = sh(returnStdout: true, script: 'date -u +%Y%m%d%H%M%S').trim()
                    env.RELEASE_DIR = "build/release"
                    env.RELEASE_TAR = "build/packages/${env.PROJECT_NAME}-${env.BUILD_TIMESTAMP}.tar.gz"
                    env.REMOTE_RELEASE_DIR = "${env.REMOTE_BASE_DIR}/${env.PROJECT_NAME}/${env.BUILD_TIMESTAMP}"
                    env.REMOTE_TEMP_DIR = "/tmp/${env.PROJECT_NAME}-${env.BUILD_TIMESTAMP}"
                }

                sh '''
                    set -euo pipefail
                    rm -rf ${RELEASE_DIR}
                    mkdir -p ${RELEASE_DIR}/public
                    cp build/bin/agent-thing ${RELEASE_DIR}/
                    cp Dockerfile ${RELEASE_DIR}/
                    cp -r frontend/dist/* ${RELEASE_DIR}/public/
                    mkdir -p build/packages
                    tar -C ${RELEASE_DIR} -czf ${RELEASE_TAR} .
                '''

                archiveArtifacts artifacts: "${env.RELEASE_TAR}", fingerprint: true
            }
        }

        stage('Deploy to pinky') {
            steps {
                withCredentials([sshUserPrivateKey(credentialsId: env.SSH_CREDENTIALS_ID, keyFileVariable: 'SSH_KEY')]) {
                    sh '''
                        set -euo pipefail
                        SSH_CMD="ssh -i ${SSH_KEY}"
                        RSYNC_CMD="rsync -az --delete -e \"${SSH_CMD}\""

                        ${SSH_CMD} ${REMOTE_USER}@${REMOTE_HOST} "sudo mkdir -p ${REMOTE_BASE_DIR}/${PROJECT_NAME}"
                        ${SSH_CMD} ${REMOTE_USER}@${REMOTE_HOST} "rm -rf ${REMOTE_TEMP_DIR} && mkdir -p ${REMOTE_TEMP_DIR}"
                        ${RSYNC_CMD} build/release/ ${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_TEMP_DIR}/
                        ${SSH_CMD} ${REMOTE_USER}@${REMOTE_HOST} "sudo mkdir -p ${REMOTE_RELEASE_DIR}/public"
                        ${SSH_CMD} ${REMOTE_USER}@${REMOTE_HOST} "sudo rsync -a ${REMOTE_TEMP_DIR}/ ${REMOTE_RELEASE_DIR}/"
                        ${SSH_CMD} ${REMOTE_USER}@${REMOTE_HOST} "sudo rm -rf ${REMOTE_TEMP_DIR}"
                        ${SSH_CMD} ${REMOTE_USER}@${REMOTE_HOST} "sudo chown -R ${REMOTE_OWNER}:${REMOTE_OWNER} ${REMOTE_RELEASE_DIR}"
                        ${SSH_CMD} ${REMOTE_USER}@${REMOTE_HOST} "sudo ln -sfn ${REMOTE_RELEASE_DIR} ${REMOTE_BASE_DIR}/${PROJECT_NAME}/current"
                        ${SSH_CMD} ${REMOTE_USER}@${REMOTE_HOST} "sudo systemctl daemon-reload"
                        ${SSH_CMD} ${REMOTE_USER}@${REMOTE_HOST} "sudo systemctl restart agent-thing.service"
                    '''
                }
            }
        }

        stage('Verify Health') {
            steps {
                sh 'curl --fail --silent --show-error --retry 5 --retry-connrefused https://agent.dev.portnumber53.com/health'
            }
        }
    }
}
