pipeline {
    agent any

    options {
        timestamps()
    }

    environment {
        // Xata / Postgres
        XATA_DATABASE_URL = credentials('prod-xata-database-url-agent-thing-truvis-co')
        XATA_API_KEY = credentials('prod-xata-api-key-agent-thing-truvis-co')
        DATABASE_URL = credentials('prod-database-url-agent-thing-truvis-co')

        // App URLs (optional; if omitted backend falls back to localhost defaults)
        // Set these as Jenkins global env vars or add credentials if you want them secret.
        APP_BASE_URL = "${env.APP_BASE_URL ?: ''}"
        BACKEND_BASE_URL = "${env.BACKEND_BASE_URL ?: ''}"

        // Google OAuth
        GOOGLE_CLIENT_ID = credentials('prod-google-client-id-agent-truvis-co')
        GOOGLE_CLIENT_SECRET = credentials('prod-google-client-secret-agent-thing-truvis-co')
        GOOGLE_REDIRECT_URL = "${env.GOOGLE_REDIRECT_URL ?: ''}"
        JWT_SECRET = credentials('prod-jwt-secret-agent-thing-truvis-co')

        // Stripe
        STRIPE_SECRET_KEY = credentials('prod-stripe-secret-key-agent-thing-truvis-co')
        STRIPE_PUBLISHABLE_KEY = credentials('prod-stripe-publishable-key-agent-thing-truvis-co')
        STRIPE_WEBHOOK_SECRET = credentials('prod-stripe-webhook-secret-agent-thing-truvis-co')
        STRIPE_PRICE_ID = "${env.STRIPE_PRICE_ID ?: ''}"

        // Cloudflare (for wrangler deploy if/when enabled)
        CLOUDFLARE_API_TOKEN = credentials('cloudflare-api-token')

        PROJECT_NAME = 'agent-thing'
    }

    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }

        stage('Backend: Test') {
            steps {
                sh 'go test ./...'
            }
        }

        stage('Backend: Build (linux/amd64 + linux/arm64)') {
            steps {
                sh '''
                    set -euo pipefail
                    mkdir -p build/bin
                    GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o build/bin/agent-thing-amd64 ./backend
                    GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/bin/agent-thing-arm64 ./backend
                '''
            }
        }

        stage('Frontend: Build') {
            steps {
                dir('frontend') {
                    sh 'npm ci'
                    sh 'npm run build'
                }
            }
        }

        stage('DB: Migrate Up') {
            steps {
                sh '''
                  set -euo pipefail
                  # Uses DATABASE_URL or XATA_DATABASE_URL from secrets env.
                  go run ./backend migrate up
                '''
            }
        }

        stage('Package Release') {
            steps {
                script {
                    env.BUILD_TS = sh(returnStdout: true, script: 'date -u +%Y%m%d%H%M%S').trim()
                    env.RELEASE_DIR = "build/release"
                    env.RELEASE_TAR = "build/packages/${env.PROJECT_NAME}-${env.BUILD_TS}.tar.gz"
                }

                sh '''
                    set -euo pipefail
                    rm -rf ${RELEASE_DIR}
                    mkdir -p ${RELEASE_DIR}/public
                    cp build/bin/agent-thing-amd64 ${RELEASE_DIR}/agent-thing-amd64
                    cp build/bin/agent-thing-arm64 ${RELEASE_DIR}/agent-thing-arm64
                    cp Dockerfile ${RELEASE_DIR}/
                    cp -r frontend/dist/* ${RELEASE_DIR}/public/
                    mkdir -p build/packages
                    tar -C ${RELEASE_DIR} -czf ${RELEASE_TAR} .
                '''

                archiveArtifacts artifacts: "${env.RELEASE_TAR}", fingerprint: true
            }
        }

        stage('Deploy: Backend (AMD64)') {
            steps {
                sh '''
                    set -euo pipefail
                    AMD_HOSTS=("192.168.68.40:22040" "192.168.68.50:22050")
                    for hp in "${AMD_HOSTS[@]}"; do
                    host="${hp%%:*}"
                    port="${hp##*:}"
                    echo "[deploy amd64] ${host}:${port}"
                    scp -P "${port}" build/bin/agent-thing-amd64 grimlock@"${host}":/tmp/agent-thing.new
                    scp -P "${port}" deploy/scripts/install_backend.sh grimlock@"${host}":/tmp/install_backend.sh
                    scp -P "${port}" deploy/config.ini.sample grimlock@"${host}":/tmp/config.ini.sample
                    scp -P "${port}" deploy/systemd/agent-thing.service grimlock@"${host}":/tmp/agent-thing.service
                    ssh -p "${port}" grimlock@"${host}" 'sudo bash -lc "set -euo pipefail; mkdir -p /opt/agent-thing/bin; install -m 0755 /tmp/agent-thing.new /opt/agent-thing/bin/agent-thing; chmod +x /tmp/install_backend.sh; /tmp/install_backend.sh"'
                    done
                '''
            }
        }

        stage('Deploy: Backend (ARM64)') {
            steps {
                sh '''
                  set -euo pipefail
                  ARM_HOSTS=(
                    "163.192.9.21:22"
                    "129.146.3.224:22"
                    "150.136.217.87:22"
                    "164.152.111.231:22"
                    "168.138.152.114:22"
                    "144.24.200.77:22"
                  )
                  for hp in "${ARM_HOSTS[@]}"; do
                    host="${hp%%:*}"
                    port="${hp##*:}"
                    echo "[deploy arm64] ${host}:${port}"
                    scp -P "${port}" build/bin/agent-thing-arm64 grimlock@"${host}":/tmp/agent-thing.new
                    scp -P "${port}" deploy/scripts/install_backend.sh grimlock@"${host}":/tmp/install_backend.sh
                    scp -P "${port}" deploy/config.ini.sample grimlock@"${host}":/tmp/config.ini.sample
                    scp -P "${port}" deploy/systemd/agent-thing.service grimlock@"${host}":/tmp/agent-thing.service
                    ssh -p "${port}" grimlock@"${host}" 'sudo bash -lc "set -euo pipefail; mkdir -p /opt/agent-thing/bin; install -m 0755 /tmp/agent-thing.new /opt/agent-thing/bin/agent-thing; chmod +x /tmp/install_backend.sh; /tmp/install_backend.sh"'
                  done
                '''
            }
        }

        stage('Deploy: Frontend (Cloudflare Workers)') {
            steps {
                dir('frontend') {
                    sh '''
                      set -euo pipefail
                      npm ci
                      npm run deploy
                    '''
                }
            }
        }
    }
}
