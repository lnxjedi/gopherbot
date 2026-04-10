import { useState } from "react";
import { 
  BookOpen, 
  Code2, 
  Terminal, 
  Copy, 
  Check, 
  ExternalLink, 
  Github, 
  Cloud, 
  Bot,
  Database,
  MessageSquare
} from "lucide-react";
import { motion, AnimatePresence } from "motion/react";
import Markdown from "react-markdown";
import { cn } from "@/src/lib/utils";

// Import files directly as raw text so we don't have to duplicate them!
import README_CONTENT from "../README.md?raw";
import TERRAFORM_MAIN from "../terraform/main.tf?raw";
import TERRAFORM_VARS from "../terraform/variables.tf?raw";
import TERRAFORM_OUTPUTS from "../terraform/outputs.tf?raw";

export default function App() {
  const [activeTab, setActiveTab] = useState<"guide" | "terraform">("guide");
  const [copied, setCopied] = useState<string | null>(null);

  const copyToClipboard = (text: string, id: string) => {
    navigator.clipboard.writeText(text);
    setCopied(id);
    setTimeout(() => setCopied(null), 2000);
  };

  return (
    <div className="min-h-screen bg-slate-50 text-slate-900 font-sans selection:bg-blue-100">
      {/* Header */}
      <header className="bg-white border-b border-slate-200 sticky top-0 z-10">
        <div className="max-w-5xl mx-auto px-6 py-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="bg-blue-600 p-2 rounded-lg">
              <Bot className="w-6 h-6 text-white" />
            </div>
            <div>
              <h1 className="text-xl font-bold tracking-tight">Gopherbot GCloud Assistant</h1>
              <p className="text-xs text-slate-500 font-medium">DevOps Chatbot Infrastructure Guide</p>
            </div>
          </div>
          <div className="flex items-center gap-4">
            <a 
              href="https://github.com/lnxjedi/gopherbot" 
              target="_blank" 
              rel="noopener noreferrer"
              className="text-slate-500 hover:text-slate-900 transition-colors"
            >
              <Github className="w-5 h-5" />
            </a>
          </div>
        </div>
      </header>

      <main className="max-w-5xl mx-auto px-6 py-12">
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-12">
          {/* Sidebar Navigation */}
          <aside className="lg:col-span-3 space-y-6">
            <nav className="space-y-1">
              <button
                onClick={() => setActiveTab("guide")}
                className={cn(
                  "w-full flex items-center gap-3 px-4 py-3 rounded-xl text-sm font-medium transition-all",
                  activeTab === "guide" 
                    ? "bg-blue-50 text-blue-700 shadow-sm" 
                    : "text-slate-600 hover:bg-slate-100"
                )}
              >
                <BookOpen className="w-4 h-4" />
                Setup Guide
              </button>
              <button
                onClick={() => setActiveTab("terraform")}
                className={cn(
                  "w-full flex items-center gap-3 px-4 py-3 rounded-xl text-sm font-medium transition-all",
                  activeTab === "terraform" 
                    ? "bg-blue-50 text-blue-700 shadow-sm" 
                    : "text-slate-600 hover:bg-slate-100"
                )}
              >
                <Code2 className="w-4 h-4" />
                Terraform Files
              </button>
            </nav>

            <div className="p-4 bg-amber-50 rounded-2xl border border-amber-100">
              <h3 className="text-xs font-bold text-amber-800 uppercase tracking-wider mb-2">Key Components</h3>
              <ul className="space-y-3">
                <li className="flex items-start gap-3">
                  <Database className="w-4 h-4 text-amber-600 mt-0.5" />
                  <span className="text-xs text-amber-900 leading-relaxed">
                    <strong>Firestore</strong>: Used as the robot's persistent "brain" for state and memory.
                  </span>
                </li>
                <li className="flex items-start gap-3">
                  <MessageSquare className="w-4 h-4 text-amber-600 mt-0.5" />
                  <span className="text-xs text-amber-900 leading-relaxed">
                    <strong>Google Chat</strong>: The primary interface for users to interact with the bot.
                  </span>
                </li>
                <li className="flex items-start gap-3">
                  <Cloud className="w-4 h-4 text-amber-600 mt-0.5" />
                  <span className="text-xs text-amber-900 leading-relaxed">
                    <strong>Pub/Sub</strong>: Enables real-time message delivery from Google Chat to the bot.
                  </span>
                </li>
              </ul>
            </div>
          </aside>

          {/* Main Content Area */}
          <div className="lg:col-span-9">
            <AnimatePresence mode="wait">
              {activeTab === "guide" ? (
                <motion.div
                  key="guide"
                  initial={{ opacity: 0, y: 10 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -10 }}
                  className="bg-white rounded-3xl border border-slate-200 shadow-sm overflow-hidden"
                >
                  <div className="p-8 lg:p-12 prose prose-slate max-w-none prose-headings:font-bold prose-h1:text-3xl prose-h2:text-xl prose-h2:mt-10 prose-h2:border-b prose-h2:pb-2 prose-code:text-blue-600 prose-code:bg-blue-50 prose-code:px-1 prose-code:rounded prose-pre:bg-slate-900 prose-pre:text-slate-100">
                    <Markdown>{README_CONTENT}</Markdown>
                  </div>
                </motion.div>
              ) : (
                <motion.div
                  key="terraform"
                  initial={{ opacity: 0, y: 10 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -10 }}
                  className="space-y-8"
                >
                  <FileBlock 
                    title="main.tf" 
                    content={TERRAFORM_MAIN} 
                    onCopy={() => copyToClipboard(TERRAFORM_MAIN, "main")}
                    isCopied={copied === "main"}
                  />
                  <FileBlock 
                    title="variables.tf" 
                    content={TERRAFORM_VARS} 
                    onCopy={() => copyToClipboard(TERRAFORM_VARS, "vars")}
                    isCopied={copied === "vars"}
                  />
                  <FileBlock 
                    title="outputs.tf" 
                    content={TERRAFORM_OUTPUTS} 
                    onCopy={() => copyToClipboard(TERRAFORM_OUTPUTS, "outputs")}
                    isCopied={copied === "outputs"}
                  />
                </motion.div>
              )}
            </AnimatePresence>
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="border-t border-slate-200 mt-24 py-12">
        <div className="max-w-5xl mx-auto px-6 flex flex-col md:flex-row items-center justify-between gap-6">
          <p className="text-sm text-slate-500">
            &copy; {new Date().getFullYear()} Gopherbot Project. Built for Google Cloud Platform.
          </p>
          <div className="flex items-center gap-6">
            <a href="https://github.com/lnxjedi/gopherbot" className="text-sm font-medium text-slate-600 hover:text-blue-600 flex items-center gap-1.5">
              Documentation <ExternalLink className="w-3 h-3" />
            </a>
            <a href="https://github.com/lnxjedi/gopherbot/issues" className="text-sm font-medium text-slate-600 hover:text-blue-600 flex items-center gap-1.5">
              Support <ExternalLink className="w-3 h-3" />
            </a>
          </div>
        </div>
      </footer>
    </div>
  );
}

function FileBlock({ title, content, onCopy, isCopied }: { title: string, content: string, onCopy: () => void, isCopied: boolean }) {
  return (
    <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
      <div className="bg-slate-50 px-6 py-3 border-b border-slate-200 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Terminal className="w-4 h-4 text-slate-400" />
          <span className="text-sm font-mono font-medium text-slate-700">{title}</span>
        </div>
        <button
          onClick={onCopy}
          className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-white border border-slate-200 text-xs font-medium text-slate-600 hover:bg-slate-50 hover:text-blue-600 transition-all active:scale-95"
        >
          {isCopied ? (
            <>
              <Check className="w-3.5 h-3.5 text-green-500" />
              Copied!
            </>
          ) : (
            <>
              <Copy className="w-3.5 h-3.5" />
              Copy Code
            </>
          )}
        </button>
      </div>
      <div className="p-0 overflow-x-auto">
        <pre className="p-6 text-sm font-mono text-slate-800 leading-relaxed">
          <code>{content}</code>
        </pre>
      </div>
    </div>
  );
}

