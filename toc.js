// Populate the sidebar
//
// This is a script, and not included directly in the page, to control the total size of the book.
// The TOC contains an entry for each page, so if each page includes a copy of the TOC,
// the total size of the page becomes O(n**2).
class MDBookSidebarScrollbox extends HTMLElement {
    constructor() {
        super();
    }
    connectedCallback() {
        this.innerHTML = '<ol class="chapter"><li class="chapter-item expanded affix "><a href="Title.html">Title</a></li><li class="chapter-item expanded affix "><a href="Status.html">Status</a></li><li class="chapter-item expanded affix "><a href="Foreword.html">Foreword</a></li><li class="chapter-item expanded affix "><a href="Introduction.html">Introduction</a></li><li class="chapter-item expanded affix "><a href="Terminology.html">Terminology</a></li><li class="chapter-item expanded affix "><li class="part-title">Part I - Getting Started</li><li class="chapter-item expanded "><a href="botdev/IDE.html"><strong aria-hidden="true">1.</strong> The Gopherbot IDE</a></li><li class="chapter-item expanded "><a href="Installation.html"><strong aria-hidden="true">2.</strong> Installing and Configuring a Gopherbot Robot</a></li><li><ol class="section"><li class="chapter-item expanded "><a href="install/LinuxInstall.html"><strong aria-hidden="true">2.1.</strong> Installation on Linux</a></li><li><ol class="section"><li class="chapter-item expanded "><a href="install/Requirements.html"><strong aria-hidden="true">2.1.1.</strong> Software Requirements</a></li><li class="chapter-item expanded "><a href="install/ManualInstall.html"><strong aria-hidden="true">2.1.2.</strong> Installing Gopherbot</a></li></ol></li><li class="chapter-item expanded "><a href="botsetup/credentials.html"><strong aria-hidden="true">2.2.</strong> Team Chat Credentials</a></li><li><ol class="section"><li class="chapter-item expanded "><a href="botsetup/slacksock.html"><strong aria-hidden="true">2.2.1.</strong> Slack Socket Mode</a></li></ol></li><li class="chapter-item expanded "><a href="RobotSetup.html"><strong aria-hidden="true">2.3.</strong> Initial Robot Setup</a></li><li><ol class="section"><li class="chapter-item expanded "><a href="botsetup/Requirements.html"><strong aria-hidden="true">2.3.1.</strong> Environment Requirements</a></li><li class="chapter-item expanded "><a href="botsetup/gopherhome.html"><strong aria-hidden="true">2.3.2.</strong> Robot Directory Structure</a></li><li class="chapter-item expanded "><a href="botsetup/Plugin.html"><strong aria-hidden="true">2.3.3.</strong> Quick Start with the Gopherbot IDE</a></li></ol></li></ol></li><li class="chapter-item expanded "><a href="RunRobot.html"><strong aria-hidden="true">3.</strong> Deploying and Running Your Robot</a></li><li><ol class="section"><li class="chapter-item expanded "><a href="deploy/deploy-environment.html"><strong aria-hidden="true">3.1.</strong> Deployment Environment Variables</a></li><li class="chapter-item expanded "><a href="deploy/Container.html"><strong aria-hidden="true">3.2.</strong> Running in a Container</a></li><li><ol class="section"><li class="chapter-item expanded "><a href="deploy/DockerDeploy.html"><strong aria-hidden="true">3.2.1.</strong> Docker Example</a></li><li class="chapter-item expanded "><a href="deploy/Kubernetes.html"><strong aria-hidden="true">3.2.2.</strong> Deploying to Kubernetes</a></li></ol></li><li class="chapter-item expanded "><a href="deploy/systemd.html"><strong aria-hidden="true">3.3.</strong> Running with Systemd</a></li></ol></li><li class="chapter-item expanded "><li class="part-title">Part II - Working with Your Robot</li><li class="chapter-item expanded "><a href="Basics.html"><strong aria-hidden="true">4.</strong> Robot Basics</a></li><li><ol class="section"><li class="chapter-item expanded "><a href="basics/ping.html"><strong aria-hidden="true">4.1.</strong> Addressing your Robot</a></li><li class="chapter-item expanded "><a href="basics/matching.html"><strong aria-hidden="true">4.2.</strong> Command Matching</a></li><li class="chapter-item expanded "><a href="basics/channels.html"><strong aria-hidden="true">4.3.</strong> Availability by Channel</a></li><li class="chapter-item expanded "><a href="basics/help.html"><strong aria-hidden="true">4.4.</strong> The built-in Help System</a></li><li class="chapter-item expanded "><a href="basics/stdplugins.html"><strong aria-hidden="true">4.5.</strong> Standard Commands</a></li><li class="chapter-item expanded "><a href="basics/context.html"><strong aria-hidden="true">4.6.</strong> Context</a></li></ol></li><li class="chapter-item expanded "><a href="Admin.html"><strong aria-hidden="true">5.</strong> Managing Your Robot</a></li><li><ol class="section"><li class="chapter-item expanded "><a href="usage/update.html"><strong aria-hidden="true">5.1.</strong> Updating from Git</a></li><li class="chapter-item expanded "><a href="extensiondev/devenv.html"><strong aria-hidden="true">5.2.</strong> Container Dev Environment</a></li><li class="chapter-item expanded "><a href="extensiondev/local.html"><strong aria-hidden="true">5.3.</strong> Local Install Dev Environment</a></li><li class="chapter-item expanded "><a href="extensiondev/CLI.html"><strong aria-hidden="true">5.4.</strong> CLI Operation</a></li><li><ol class="section"><li class="chapter-item expanded "><a href="extensiondev/secrets.html"><strong aria-hidden="true">5.4.1.</strong> Encrypting Secrets</a></li></ol></li><li class="chapter-item expanded "><a href="extensiondev/terminal.html"><strong aria-hidden="true">5.5.</strong> Using the Terminal Connector</a></li><li class="chapter-item expanded "><a href="usage/admin.html"><strong aria-hidden="true">5.6.</strong> Administrator Commands</a></li><li class="chapter-item expanded "><a href="usage/logging.html"><strong aria-hidden="true">5.7.</strong> Logging</a></li></ol></li><li class="chapter-item expanded "><li class="part-title">Part III - Worked Examples</li><li class="chapter-item expanded "><a href="customizing/first-plugin.html"><strong aria-hidden="true">6.</strong> Writing Your First Plugin</a></li><li class="chapter-item expanded "><a href="customizing.html"><strong aria-hidden="true">7.</strong> Writing Custom Extensions for Your Robot</a></li><li><ol class="section"><li class="chapter-item expanded "><a href="customizing/style.html"><strong aria-hidden="true">7.1.</strong> Style Guide</a></li><li><ol class="section"><li class="chapter-item expanded "><a href="customizing/syntax-help.html"><strong aria-hidden="true">7.1.1.</strong> Help for Invalid Command Syntax</a></li></ol></li></ol></li><li class="chapter-item expanded "><a href="pipelines/integrations.html"><strong aria-hidden="true">8.</strong> Gopherbot Tool Integrations</a></li><li><ol class="section"><li class="chapter-item expanded "><a href="pipelines/ssh.html"><strong aria-hidden="true">8.1.</strong> Integrating with SSH</a></li></ol></li><li class="chapter-item expanded "><li class="part-title">Part IV - Reference</li><li class="chapter-item expanded "><a href="Configuration.html"><strong aria-hidden="true">9.</strong> Gopherbot Configuration Reference</a></li><li><ol class="section"><li class="chapter-item expanded "><a href="config/file.html"><strong aria-hidden="true">9.1.</strong> Configuration File Loading</a></li><li class="chapter-item expanded "><a href="config/job-plug.html"><strong aria-hidden="true">9.2.</strong> Job and Plugin Configuration</a></li><li class="chapter-item expanded "><a href="config/troubleshooting.html"><strong aria-hidden="true">9.3.</strong> Troubleshooting</a></li></ol></li><li class="chapter-item expanded "><a href="api/API-Introduction.html"><strong aria-hidden="true">10.</strong> Gopherbot Scripting API</a></li><li><ol class="section"><li class="chapter-item expanded "><a href="Environment-Variables.html"><strong aria-hidden="true">10.1.</strong> Script Environment Variables</a></li><li class="chapter-item expanded "><a href="api/Languages.html"><strong aria-hidden="true">10.2.</strong> Language Templates</a></li><li class="chapter-item expanded "><a href="api/Attribute-Retrieval-API.html"><strong aria-hidden="true">10.3.</strong> Attribute Retrieval</a></li><li class="chapter-item expanded "><a href="api/Brain-API.html"><strong aria-hidden="true">10.4.</strong> Brain Methods</a></li><li class="chapter-item expanded "><a href="api/Message-Sending-API.html"><strong aria-hidden="true">10.5.</strong> Message Sending</a></li><li class="chapter-item expanded "><a href="api/Pipeline-API.html"><strong aria-hidden="true">10.6.</strong> Pipeline Construction</a></li><li class="chapter-item expanded "><a href="api/Response-Request-API.html"><strong aria-hidden="true">10.7.</strong> Requesting Responses</a></li><li class="chapter-item expanded "><a href="api/Utility-API.html"><strong aria-hidden="true">10.8.</strong> Utility</a></li></ol></li><li class="chapter-item expanded "><a href="pipelines/jobspipes.html"><strong aria-hidden="true">11.</strong> Pipelines, Plugins, Jobs and Tasks</a></li><li><ol class="section"><li class="chapter-item expanded "><a href="pipelines/primary.html"><strong aria-hidden="true">11.1.</strong> The Primary Pipeline</a></li><li class="chapter-item expanded "><a href="pipelines/final.html"><strong aria-hidden="true">11.2.</strong> The Final Pipeline</a></li><li class="chapter-item expanded "><a href="pipelines/fail.html"><strong aria-hidden="true">11.3.</strong> The Fail Pipeline</a></li><li class="chapter-item expanded "><a href="pipelines/TaskEnvironment.html"><strong aria-hidden="true">11.4.</strong> Task Environment Variables</a></li><li class="chapter-item expanded "><a href="pipelines/tasks.html"><strong aria-hidden="true">11.5.</strong> All Included Tasks</a></li></ol></li><li class="chapter-item expanded "><li class="part-title">Appendix</li><li class="chapter-item expanded "><a href="appendices/Appendix.html"><strong aria-hidden="true">12.</strong> Appendix</a></li><li><ol class="section"><li class="chapter-item expanded "><a href="appendices/InstallArchive.html"><strong aria-hidden="true">12.1.</strong> A - Gopherbot Install Archive</a></li><li class="chapter-item expanded "><a href="appendices/Protocols.html"><strong aria-hidden="true">12.2.</strong> B - Protocols</a></li><li><ol class="section"><li class="chapter-item expanded "><a href="appendices/slack.html"><strong aria-hidden="true">12.2.1.</strong> B.1 - Slack</a></li><li class="chapter-item expanded "><a href="appendices/rocket.html"><strong aria-hidden="true">12.2.2.</strong> B.2 - Rocket.Chat</a></li><li class="chapter-item expanded "><a href="appendices/terminal.html"><strong aria-hidden="true">12.2.3.</strong> B.3 - Terminal</a></li><li class="chapter-item expanded "><a href="appendices/testproto.html"><strong aria-hidden="true">12.2.4.</strong> B.4 - Test</a></li><li class="chapter-item expanded "><a href="appendices/nullconn.html"><strong aria-hidden="true">12.2.5.</strong> B.5 - Nullconn</a></li></ol></li></ol></li></ol>';
        // Set the current, active page, and reveal it if it's hidden
        let current_page = document.location.href.toString();
        if (current_page.endsWith("/")) {
            current_page += "index.html";
        }
        var links = Array.prototype.slice.call(this.querySelectorAll("a"));
        var l = links.length;
        for (var i = 0; i < l; ++i) {
            var link = links[i];
            var href = link.getAttribute("href");
            if (href && !href.startsWith("#") && !/^(?:[a-z+]+:)?\/\//.test(href)) {
                link.href = path_to_root + href;
            }
            // The "index" page is supposed to alias the first chapter in the book.
            if (link.href === current_page || (i === 0 && path_to_root === "" && current_page.endsWith("/index.html"))) {
                link.classList.add("active");
                var parent = link.parentElement;
                if (parent && parent.classList.contains("chapter-item")) {
                    parent.classList.add("expanded");
                }
                while (parent) {
                    if (parent.tagName === "LI" && parent.previousElementSibling) {
                        if (parent.previousElementSibling.classList.contains("chapter-item")) {
                            parent.previousElementSibling.classList.add("expanded");
                        }
                    }
                    parent = parent.parentElement;
                }
            }
        }
        // Track and set sidebar scroll position
        this.addEventListener('click', function(e) {
            if (e.target.tagName === 'A') {
                sessionStorage.setItem('sidebar-scroll', this.scrollTop);
            }
        }, { passive: true });
        var sidebarScrollTop = sessionStorage.getItem('sidebar-scroll');
        sessionStorage.removeItem('sidebar-scroll');
        if (sidebarScrollTop) {
            // preserve sidebar scroll position when navigating via links within sidebar
            this.scrollTop = sidebarScrollTop;
        } else {
            // scroll sidebar to current active section when navigating via "next/previous chapter" buttons
            var activeSection = document.querySelector('#sidebar .active');
            if (activeSection) {
                activeSection.scrollIntoView({ block: 'center' });
            }
        }
        // Toggle buttons
        var sidebarAnchorToggles = document.querySelectorAll('#sidebar a.toggle');
        function toggleSection(ev) {
            ev.currentTarget.parentElement.classList.toggle('expanded');
        }
        Array.from(sidebarAnchorToggles).forEach(function (el) {
            el.addEventListener('click', toggleSection);
        });
    }
}
window.customElements.define("mdbook-sidebar-scrollbox", MDBookSidebarScrollbox);
