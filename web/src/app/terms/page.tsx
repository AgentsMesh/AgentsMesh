"use client";

import Link from "next/link";
import { PageHeader } from "@/components/common";

export default function TermsPage() {
  return (
    <div className="min-h-screen bg-background">
      <PageHeader />

      {/* Content */}
      <main className="container mx-auto px-4 py-12 max-w-4xl">
        <h1 className="text-4xl font-bold mb-8">Terms of Service</h1>
        <p className="text-muted-foreground mb-8">Last updated: January 2025</p>

        <div className="bg-muted border border-border rounded-lg p-4 mb-8 text-sm text-muted-foreground">
          This page is currently available in English only. / 此页面目前仅提供英文版本。
        </div>

        <div className="prose prose-neutral dark:prose-invert max-w-none space-y-8">
          <section>
            <h2 className="text-2xl font-semibold mb-4">1. Acceptance of Terms</h2>
            <p className="text-muted-foreground leading-relaxed">
              By accessing or using AgentsMesh (the &quot;Service&quot;) provided by AgentsMesh, Inc. (&quot;AgentsMesh,&quot; &quot;we,&quot; &quot;our,&quot; or &quot;us&quot;), you agree to be bound by these Terms of Service (&quot;Terms&quot;). If you do not agree to these Terms, you may not access or use the Service. If you are using the Service on behalf of an organization, you represent and warrant that you have the authority to bind that organization to these Terms.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">2. Description of Service</h2>
            <p className="text-muted-foreground leading-relaxed mb-4">
              AgentsMesh is a multi-agent AI code collaboration platform that provides:
            </p>
            <ul className="list-disc list-inside text-muted-foreground space-y-2">
              <li>AgentPod: Remote AI workstations for running AI coding agents</li>
              <li>AgentsMesh Channels: Communication infrastructure for multi-agent collaboration</li>
              <li>Ticket Management: Task tracking integrated with AI agents</li>
              <li>Self-Hosted Runners: Infrastructure for running agents in your own environment</li>
              <li>Repository Integration: Connections to GitHub, GitLab, Gitee, and other Git providers</li>
            </ul>
            <p className="text-muted-foreground leading-relaxed mt-4">
              The Service supports various AI coding agents including, but not limited to, Claude Code, Codex CLI, Gemini CLI, Aider, and custom agent implementations.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">3. Account Registration</h2>
            <p className="text-muted-foreground leading-relaxed mb-4">
              To use certain features of the Service, you must create an account. You agree to:
            </p>
            <ul className="list-disc list-inside text-muted-foreground space-y-2">
              <li>Provide accurate, current, and complete information during registration</li>
              <li>Maintain and update your information to keep it accurate</li>
              <li>Maintain the security of your account credentials</li>
              <li>Notify us immediately of any unauthorized access to your account</li>
              <li>Be responsible for all activities that occur under your account</li>
            </ul>
            <p className="text-muted-foreground leading-relaxed mt-4">
              You must be at least 16 years old to create an account. We reserve the right to refuse service, terminate accounts, or remove content at our sole discretion.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">4. Subscription and Payments</h2>

            <h3 className="text-xl font-medium mb-3 mt-6">4.1 Plans and Pricing</h3>
            <p className="text-muted-foreground leading-relaxed">
              The Service is offered under various subscription plans, including free and paid tiers. Current pricing and plan features are available on our website. We reserve the right to modify pricing with 30 days&apos; notice.
            </p>

            <h3 className="text-xl font-medium mb-3 mt-6">4.2 Payment Terms</h3>
            <p className="text-muted-foreground leading-relaxed">
              For paid plans, you agree to pay all applicable fees. Payments are processed by third-party payment providers. Subscriptions automatically renew unless cancelled before the renewal date. Refunds are provided in accordance with our refund policy.
            </p>

            <h3 className="text-xl font-medium mb-3 mt-6">4.3 BYOK (Bring Your Own Key)</h3>
            <p className="text-muted-foreground leading-relaxed">
              The Service operates on a BYOK model where you provide your own AI provider API keys. You are responsible for all charges incurred with your AI providers (Anthropic, OpenAI, Google, etc.). AgentsMesh does not control or limit your usage of external AI services.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">5. Acceptable Use</h2>
            <p className="text-muted-foreground leading-relaxed mb-4">
              You agree not to use the Service to:
            </p>
            <ul className="list-disc list-inside text-muted-foreground space-y-2">
              <li>Violate any applicable laws, regulations, or third-party rights</li>
              <li>Infringe upon intellectual property rights of others</li>
              <li>Create, distribute, or store malware, viruses, or malicious code</li>
              <li>Attempt to gain unauthorized access to systems, networks, or data</li>
              <li>Interfere with or disrupt the Service or its infrastructure</li>
              <li>Engage in any activity that could harm, disable, or impair the Service</li>
              <li>Use the Service to harass, abuse, or harm others</li>
              <li>Circumvent usage limits, security measures, or access controls</li>
              <li>Resell, sublicense, or redistribute the Service without authorization</li>
              <li>Use the Service for cryptocurrency mining without explicit permission</li>
              <li>Generate content that violates AI provider acceptable use policies</li>
            </ul>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">6. API Keys and Credentials</h2>
            <p className="text-muted-foreground leading-relaxed mb-4">
              You are solely responsible for:
            </p>
            <ul className="list-disc list-inside text-muted-foreground space-y-2">
              <li>Securing and protecting your API keys and credentials</li>
              <li>Any charges incurred through the use of your API keys</li>
              <li>Ensuring your use of third-party APIs complies with their terms of service</li>
              <li>Rotating keys if you suspect unauthorized access</li>
            </ul>
            <p className="text-muted-foreground leading-relaxed mt-4">
              AgentsMesh encrypts stored credentials but is not liable for any unauthorized access resulting from your failure to protect your credentials or from security breaches beyond our reasonable control.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">7. Intellectual Property</h2>

            <h3 className="text-xl font-medium mb-3 mt-6">7.1 Your Content</h3>
            <p className="text-muted-foreground leading-relaxed">
              You retain all rights to the code, content, and materials you create or upload through the Service (&quot;Your Content&quot;). By using the Service, you grant us a limited license to host, store, and process Your Content solely to provide the Service.
            </p>

            <h3 className="text-xl font-medium mb-3 mt-6">7.2 Our Service</h3>
            <p className="text-muted-foreground leading-relaxed">
              The Service, including its software, design, features, and documentation, is owned by AgentsMesh and protected by intellectual property laws. You may not copy, modify, distribute, sell, or lease any part of our Service without written permission.
            </p>

            <h3 className="text-xl font-medium mb-3 mt-6">7.3 Feedback</h3>
            <p className="text-muted-foreground leading-relaxed">
              If you provide feedback, suggestions, or ideas about the Service, you grant us the right to use such feedback without restriction or compensation to you.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">8. Self-Hosted Runners</h2>
            <p className="text-muted-foreground leading-relaxed mb-4">
              If you use self-hosted runners:
            </p>
            <ul className="list-disc list-inside text-muted-foreground space-y-2">
              <li>You are responsible for the security and maintenance of your infrastructure</li>
              <li>You must ensure compliance with applicable laws regarding data processing</li>
              <li>AgentsMesh is not responsible for data processed on your infrastructure</li>
              <li>You must keep runner software updated to maintain compatibility</li>
            </ul>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">9. Third-Party Services</h2>
            <p className="text-muted-foreground leading-relaxed">
              The Service integrates with third-party services including AI providers (Anthropic, OpenAI, Google) and Git providers (GitHub, GitLab, Gitee). Your use of these services is subject to their respective terms of service. AgentsMesh is not responsible for the availability, accuracy, or content of third-party services.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">10. Service Availability</h2>
            <p className="text-muted-foreground leading-relaxed">
              We strive to maintain high availability but do not guarantee uninterrupted access to the Service. We may perform scheduled maintenance with advance notice when possible. We are not liable for any interruption, delay, or failure of the Service due to circumstances beyond our reasonable control.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">11. Disclaimer of Warranties</h2>
            <p className="text-muted-foreground leading-relaxed">
              THE SERVICE IS PROVIDED &quot;AS IS&quot; AND &quot;AS AVAILABLE&quot; WITHOUT WARRANTIES OF ANY KIND, WHETHER EXPRESS, IMPLIED, STATUTORY, OR OTHERWISE. WE SPECIFICALLY DISCLAIM ALL IMPLIED WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE, TITLE, AND NON-INFRINGEMENT. WE DO NOT WARRANT THAT THE SERVICE WILL BE UNINTERRUPTED, ERROR-FREE, OR SECURE, OR THAT DEFECTS WILL BE CORRECTED.
            </p>
            <p className="text-muted-foreground leading-relaxed mt-4">
              WE DO NOT GUARANTEE THE ACCURACY, RELIABILITY, OR QUALITY OF ANY AI-GENERATED CONTENT OR CODE. YOU ARE SOLELY RESPONSIBLE FOR REVIEWING AND VALIDATING ALL AI-GENERATED OUTPUT BEFORE USE.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">12. Limitation of Liability</h2>
            <p className="text-muted-foreground leading-relaxed">
              TO THE MAXIMUM EXTENT PERMITTED BY LAW, AGENTSMESH SHALL NOT BE LIABLE FOR ANY INDIRECT, INCIDENTAL, SPECIAL, CONSEQUENTIAL, OR PUNITIVE DAMAGES, INCLUDING BUT NOT LIMITED TO LOSS OF PROFITS, DATA, USE, GOODWILL, OR OTHER INTANGIBLE LOSSES, RESULTING FROM:
            </p>
            <ul className="list-disc list-inside text-muted-foreground space-y-2 mt-4">
              <li>Your access to or use of (or inability to access or use) the Service</li>
              <li>Any conduct or content of any third party on the Service</li>
              <li>Any content obtained from the Service</li>
              <li>Unauthorized access, use, or alteration of your transmissions or content</li>
              <li>AI-generated code or content that contains errors or causes harm</li>
              <li>Charges incurred through your AI provider API keys</li>
            </ul>
            <p className="text-muted-foreground leading-relaxed mt-4">
              IN NO EVENT SHALL OUR TOTAL LIABILITY TO YOU EXCEED THE AMOUNT YOU PAID US IN THE TWELVE (12) MONTHS PRECEDING THE CLAIM, OR ONE HUNDRED DOLLARS ($100), WHICHEVER IS GREATER.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">13. Indemnification</h2>
            <p className="text-muted-foreground leading-relaxed">
              You agree to indemnify, defend, and hold harmless AgentsMesh, its officers, directors, employees, and agents from and against any claims, liabilities, damages, losses, and expenses, including reasonable attorney&apos;s fees, arising out of or in any way connected with your access to or use of the Service, your violation of these Terms, or your violation of any third-party rights.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">14. Termination</h2>
            <p className="text-muted-foreground leading-relaxed mb-4">
              We may suspend or terminate your access to the Service at any time, with or without cause, with or without notice. Upon termination:
            </p>
            <ul className="list-disc list-inside text-muted-foreground space-y-2">
              <li>Your right to use the Service will immediately cease</li>
              <li>You may request export of your data within 30 days</li>
              <li>We may delete your data after a reasonable retention period</li>
              <li>You remain liable for any fees incurred before termination</li>
            </ul>
            <p className="text-muted-foreground leading-relaxed mt-4">
              You may terminate your account at any time through your account settings or by contacting us.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">15. Modifications to Terms</h2>
            <p className="text-muted-foreground leading-relaxed">
              We reserve the right to modify these Terms at any time. We will provide notice of material changes by posting the updated Terms on our website and updating the &quot;Last updated&quot; date. Your continued use of the Service after changes become effective constitutes acceptance of the modified Terms.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">16. Governing Law and Disputes</h2>
            <p className="text-muted-foreground leading-relaxed mb-4">
              These Terms shall be governed by and construed in accordance with the laws of the State of Delaware, United States, without regard to its conflict of law provisions.
            </p>
            <p className="text-muted-foreground leading-relaxed">
              Any disputes arising from these Terms or your use of the Service shall be resolved through binding arbitration in accordance with the rules of the American Arbitration Association. You agree to waive your right to participate in class action lawsuits or class-wide arbitration.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">17. Severability</h2>
            <p className="text-muted-foreground leading-relaxed">
              If any provision of these Terms is held to be unenforceable, the remaining provisions shall continue in full force and effect. The unenforceable provision shall be modified to the minimum extent necessary to make it enforceable while preserving its intent.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">18. Entire Agreement</h2>
            <p className="text-muted-foreground leading-relaxed">
              These Terms, together with our Privacy Policy, constitute the entire agreement between you and AgentsMesh regarding the Service and supersede all prior agreements and understandings.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">19. Contact Information</h2>
            <p className="text-muted-foreground leading-relaxed">
              If you have any questions about these Terms of Service, please contact us at:
            </p>
            <div className="mt-4 p-4 bg-muted rounded-lg text-muted-foreground">
              <p><strong>AgentsMesh, Inc.</strong></p>
              <p>Email:{" "}
                <a href="mailto:legal@agentsmesh.ai" className="text-primary hover:underline">
                  legal@agentsmesh.ai
                </a>
              </p>
            </div>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">20. Export Compliance</h2>
            <p className="text-muted-foreground leading-relaxed">
              The Service may be subject to export control laws. You agree to comply with all applicable export and re-export control laws and regulations and not to transfer, export, or re-export the Service to any prohibited destination, entity, or person.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">21. Force Majeure</h2>
            <p className="text-muted-foreground leading-relaxed">
              AgentsMesh shall not be liable for any failure or delay in performing its obligations under these Terms due to circumstances beyond its reasonable control, including but not limited to acts of God, natural disasters, war, terrorism, riots, embargoes, acts of civil or military authorities, fire, floods, accidents, pandemics, strikes, or shortages of transportation, facilities, fuel, energy, labor, or materials.
            </p>
          </section>
        </div>
      </main>

      {/* Footer */}
      <footer className="border-t border-border mt-16">
        <div className="container mx-auto px-4 py-8">
          <div className="flex flex-col md:flex-row justify-between items-center gap-4">
            <p className="text-sm text-muted-foreground">
              &copy; {new Date().getFullYear()} AgentsMesh, Inc. All rights reserved.
            </p>
            <div className="flex gap-6">
              <Link href="/privacy" className="text-sm text-muted-foreground hover:text-foreground">
                Privacy Policy
              </Link>
              <Link href="/terms" className="text-sm text-muted-foreground hover:text-foreground">
                Terms of Service
              </Link>
              <Link href="/docs" className="text-sm text-muted-foreground hover:text-foreground">
                Documentation
              </Link>
            </div>
          </div>
        </div>
      </footer>
    </div>
  );
}
