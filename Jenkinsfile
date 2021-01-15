@Library('dst-shared@master') _

dockerBuildPipeline {
        githubPushRepo = "Cray-HPE/hms-discovery"
   repository = "cray"
   imagePrefix = "hms"
   app = "discovery"
   name = "hms-discovery"
   description = "HMS image for hardware discovery"
   dockerfile = "Dockerfile"
   slackNotification = ["", "", false, false, true, true]
   product = "csm"
}
