@Library('dst-shared@release/shasta-1.4') _

dockerBuildPipeline {
   repository = "cray"
   imagePrefix = "hms"
   app = "discovery"
   name = "hms-discovery"
   description = "HMS image for hardware discovery"
   dockerfile = "Dockerfile"
   slackNotification = ["", "", false, false, true, true]
   product = "csm"
}
