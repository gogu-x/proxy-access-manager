// 与 ops 一致：分别构建 Go 后端和 Vue 前端镜像，可选推送到镜像仓库。
pipeline {
    agent any

    parameters {
        string(name: 'IMAGE_TAG', defaultValue: '', description: '镜像标签；留空时使用当前分支和短提交号')
        string(name: 'DOCKER_REGISTRY', defaultValue: 'docker.io', description: '镜像仓库地址')
        string(name: 'DOCKER_NAMESPACE', defaultValue: 'xiaoguyun', description: '镜像仓库命名空间')
        booleanParam(name: 'PUSH', defaultValue: false, description: '构建成功后推送镜像')
    }

    options {
        timestamps()
        disableConcurrentBuilds()
        timeout(time: 40, unit: 'MINUTES')
    }

    environment {
        DOCKERHUB_CREDENTIALS = 'dockerhub-xiaoguyun'
        BACKEND_IMAGE_NAME = 'proxy-access-manager-backend'
        FRONTEND_IMAGE_NAME = 'proxy-access-manager-web'
    }

    stages {
        stage('Checkout') {
            steps {
                checkout scm
                script {
                    def sha = sh(script: 'git rev-parse --short HEAD', returnStdout: true).trim()
                    def branch = env.BRANCH_NAME ?: sh(script: 'git branch --show-current', returnStdout: true).trim()
                    def safeBranch = branch.replaceAll(/[^A-Za-z0-9_.-]/, '-')
                    env.RESOLVED_TAG = params.IMAGE_TAG?.trim() ? params.IMAGE_TAG.trim() : "${safeBranch ?: 'build'}-${sha}"
                    env.BACKEND_IMAGE = "${params.DOCKER_REGISTRY}/${params.DOCKER_NAMESPACE}/${env.BACKEND_IMAGE_NAME}:${env.RESOLVED_TAG}"
                    env.FRONTEND_IMAGE = "${params.DOCKER_REGISTRY}/${params.DOCKER_NAMESPACE}/${env.FRONTEND_IMAGE_NAME}:${env.RESOLVED_TAG}"
                    echo "Backend image : ${env.BACKEND_IMAGE}"
                    echo "Frontend image: ${env.FRONTEND_IMAGE}"
                }
            }
        }

        stage('Build backend') {
            steps { sh 'docker build -f server/Dockerfile -t "$BACKEND_IMAGE" server' }
        }
        stage('Build frontend') {
            steps { sh 'docker build -f frontend/Dockerfile -t "$FRONTEND_IMAGE" frontend' }
        }
        stage('Push') {
            when { expression { return params.PUSH } }
            steps {
                withCredentials([usernamePassword(
                    credentialsId: "${DOCKERHUB_CREDENTIALS}",
                    usernameVariable: 'REGISTRY_USER',
                    passwordVariable: 'REGISTRY_PASS'
                )]) {
                    sh '''
                        echo "$REGISTRY_PASS" | docker login "$DOCKER_REGISTRY" -u "$REGISTRY_USER" --password-stdin
                        docker push "$BACKEND_IMAGE"
                        docker push "$FRONTEND_IMAGE"
                        docker logout "$DOCKER_REGISTRY"
                    '''
                }
            }
        }
    }

    post {
        success {
            echo "Built ${env.BACKEND_IMAGE} and ${env.FRONTEND_IMAGE}" +
                 (params.PUSH ? " and pushed to ${params.DOCKER_REGISTRY}." : " (not pushed).")
        }
        always {
            sh 'docker image rm "$BACKEND_IMAGE" "$FRONTEND_IMAGE" 2>/dev/null || true'
        }
    }
}
