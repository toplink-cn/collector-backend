pipeline {
    agent any;
    
    tools { go '1.20.3' }
    
    stages {
        stage('pull'){
            steps {
                git branch: 'main', credentialsId: 'github_deploy', url: 'git@github.com:toplink-cn/collector-backend.git'
            }
        }
        stage('Build'){
            steps {
                sh "GOOS=linux GOARCH=amd64 go build -o collector"
            }
        }
        stage('Packing Image') {
            steps {
                sh '''
docker build --rm -t prod-registry.toplinksoftware.com/dcim/collector-backend ./
docker save prod-registry.toplinksoftware.com/dcim/collector-backend -o dcim-collector-backend.img
'''
            }
        }
        stage('Archive Artifacts'){
            steps {
                archiveArtifacts artifacts: 'dcim-collector-backend.img'
            }
        }
        stage('Release'){
            steps {
                sh 'docker push prod-registry.toplinksoftware.com/dcim/collector-backend'
            }
        }
    }
    
    post {
        always {
            deleteDir()
        }
    }
}