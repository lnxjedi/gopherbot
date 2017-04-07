import os

class Robot:
    "Instantiate a robot object for use with Gopherbot"
    def __init__(self):
        self.channel = os.getenv("GOPHER_CHANNEL")
        self.user = os.getenv("GOPHER_USER")
        self.plugin_id = os.getenv("GOPHER_PLUGIN_ID")

class DirectBot(Robot):
    "Instantiate a robot for direct messaging with the user"
    def __init__(self):
        self.channel = ""
        self.user = os.getenv("GOPHER_USER")
        self.plugin_id = os.getenv("GOPHER_PLUGIN_ID")
        
