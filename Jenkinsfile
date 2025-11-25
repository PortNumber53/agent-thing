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

        // Google OAuth
        GOOGLE_CLIENT_ID = credentials('prod-google-client-id-agent-truvis-co')
        GOOGLE_CLIENT_SECRET = credentials('prod-google-client-secret-agent-thing-truvis-co')
        JWT_SECRET = credentials('prod-jwt-secret-agent-thing-truvis-co')

        // Stripe
        STRIPE_SECRET_KEY = credentials('prod-stripe-secret-key-agent-thing-truvis-co')
        STRIPE_PUBLISHABLE_KEY = credentials('prod-stripe-publishable-key-agent-thing-truvis-co')
        STRIPE_WEBHOOK_SECRET = credentials('prod-stripe-webhook-secret-agent-thing-truvis-co')

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

        stage('Backend: Build') {
            steps {
                sh '''
                  set -euo pipefail
                  mkdir -p build/bin
                  go build -o build/bin/agent-thing ./backend
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
                  cp build/bin/agent-thing ${RELEASE_DIR}/
                  cp Dockerfile ${RELEASE_DIR}/
                  cp -r frontend/dist/* ${RELEASE_DIR}/public/
                  mkdir -p build/packages
                  tar -C ${RELEASE_DIR} -czf ${RELEASE_TAR} .
                '''

                archiveArtifacts artifacts: "${env.RELEASE_TAR}", fingerprint: true
            }
        }
    }
}
