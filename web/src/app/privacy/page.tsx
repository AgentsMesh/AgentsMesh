"use client";

import Link from "next/link";
import { PageHeader } from "@/components/common";

export default function PrivacyPage() {
  return (
    <div className="min-h-screen bg-background">
      <PageHeader />

      {/* Content */}
      <main className="container mx-auto px-4 py-12 max-w-4xl">
        <h1 className="text-4xl font-bold mb-8">Privacy Policy</h1>
        <p className="text-muted-foreground mb-8">Last updated: January 2025</p>

        <div className="bg-muted border border-border rounded-lg p-4 mb-8 text-sm text-muted-foreground">
          This page is currently available in English only. / 此页面目前仅提供英文版本。
        </div>

        <div className="prose prose-neutral dark:prose-invert max-w-none space-y-8">
          <section>
            <h2 className="text-2xl font-semibold mb-4">1. Introduction</h2>
            <p className="text-muted-foreground leading-relaxed">
              AgentsMesh, Inc. (&quot;AgentsMesh,&quot; &quot;we,&quot; &quot;our,&quot; or &quot;us&quot;) is committed to protecting your privacy. This Privacy Policy explains how we collect, use, disclose, and safeguard your information when you use our multi-agent AI code collaboration platform (the &quot;Service&quot;). Please read this privacy policy carefully. If you do not agree with the terms of this privacy policy, please do not access the Service.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">2. Information We Collect</h2>

            <h3 className="text-xl font-medium mb-3 mt-6">2.1 Personal Information You Provide</h3>
            <p className="text-muted-foreground leading-relaxed mb-4">
              We collect information you voluntarily provide when registering for the Service, including:
            </p>
            <ul className="list-disc list-inside text-muted-foreground space-y-2">
              <li>Name and email address</li>
              <li>Username and password (hashed and salted)</li>
              <li>Profile information (avatar, preferences)</li>
              <li>Payment information (processed by third-party payment processors)</li>
              <li>Organization and team information</li>
            </ul>

            <h3 className="text-xl font-medium mb-3 mt-6">2.2 API Credentials and Secrets</h3>
            <p className="text-muted-foreground leading-relaxed mb-4">
              To enable AI agent functionality, you may provide:
            </p>
            <ul className="list-disc list-inside text-muted-foreground space-y-2">
              <li>AI provider API keys (Anthropic, OpenAI, Google)</li>
              <li>Git provider access tokens and SSH keys</li>
              <li>Webhook secrets and integration credentials</li>
            </ul>
            <p className="text-muted-foreground leading-relaxed mt-4">
              <strong>All API credentials are encrypted at rest using AES-256 encryption and are never stored in plaintext.</strong>
            </p>

            <h3 className="text-xl font-medium mb-3 mt-6">2.3 Code and Content</h3>
            <p className="text-muted-foreground leading-relaxed mb-4">
              When using the Service, we may process:
            </p>
            <ul className="list-disc list-inside text-muted-foreground space-y-2">
              <li>Source code you access through connected repositories</li>
              <li>AI agent conversations and outputs</li>
              <li>Messages in collaboration channels</li>
              <li>Ticket descriptions and content</li>
            </ul>
            <p className="text-muted-foreground leading-relaxed mt-4">
              <strong>For self-hosted runner deployments, your code remains entirely within your infrastructure and is never transmitted to AgentsMesh servers.</strong>
            </p>

            <h3 className="text-xl font-medium mb-3 mt-6">2.4 Automatically Collected Information</h3>
            <ul className="list-disc list-inside text-muted-foreground space-y-2">
              <li>Device information (browser type, operating system)</li>
              <li>IP address and geographic location (country/region level)</li>
              <li>Usage patterns and feature interactions</li>
              <li>Error logs and performance metrics</li>
              <li>Cookies and similar tracking technologies</li>
            </ul>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">3. How We Use Your Information</h2>
            <p className="text-muted-foreground leading-relaxed mb-4">
              We use the information we collect to:
            </p>
            <ul className="list-disc list-inside text-muted-foreground space-y-2">
              <li>Provide, operate, and maintain the Service</li>
              <li>Process transactions and send related information</li>
              <li>Send administrative information, updates, and security alerts</li>
              <li>Respond to inquiries and provide customer support</li>
              <li>Improve and personalize the Service</li>
              <li>Monitor and analyze usage trends and preferences</li>
              <li>Detect, prevent, and address technical issues and security threats</li>
              <li>Comply with legal obligations</li>
            </ul>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">4. Data Sharing and Disclosure</h2>
            <p className="text-muted-foreground leading-relaxed mb-4">
              <strong>We do not sell, trade, or rent your personal information to third parties.</strong> We may share information in the following circumstances:
            </p>

            <h3 className="text-xl font-medium mb-3 mt-6">4.1 Service Providers</h3>
            <p className="text-muted-foreground leading-relaxed">
              We share information with third-party vendors who provide services on our behalf, including cloud hosting, payment processing, and analytics. These providers are contractually obligated to protect your information.
            </p>

            <h3 className="text-xl font-medium mb-3 mt-6">4.2 AI and Git Providers</h3>
            <p className="text-muted-foreground leading-relaxed">
              When you use AI agents or connect Git repositories, your API keys are used to authenticate with these services on your behalf. We do not share your keys with any other parties.
            </p>

            <h3 className="text-xl font-medium mb-3 mt-6">4.3 Legal Requirements</h3>
            <p className="text-muted-foreground leading-relaxed">
              We may disclose information if required by law, court order, or government regulation, or if we believe disclosure is necessary to protect our rights, your safety, or the safety of others.
            </p>

            <h3 className="text-xl font-medium mb-3 mt-6">4.4 Business Transfers</h3>
            <p className="text-muted-foreground leading-relaxed">
              In the event of a merger, acquisition, or sale of assets, your information may be transferred. We will provide notice before your information becomes subject to a different privacy policy.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">5. Data Security</h2>
            <p className="text-muted-foreground leading-relaxed mb-4">
              We implement comprehensive security measures to protect your information:
            </p>
            <ul className="list-disc list-inside text-muted-foreground space-y-2">
              <li><strong>Encryption in Transit:</strong> All data transmitted to and from our servers uses TLS 1.3 encryption</li>
              <li><strong>Encryption at Rest:</strong> Sensitive data, including API keys and passwords, is encrypted using AES-256</li>
              <li><strong>Access Controls:</strong> Strict role-based access controls limit employee access to customer data</li>
              <li><strong>Infrastructure Security:</strong> Our infrastructure is hosted in SOC 2 compliant data centers</li>
              <li><strong>Regular Audits:</strong> We conduct regular security assessments and penetration testing</li>
              <li><strong>Incident Response:</strong> We maintain incident response procedures to address security events</li>
            </ul>
            <p className="text-muted-foreground leading-relaxed mt-4">
              While we strive to protect your information, no method of transmission over the Internet or electronic storage is 100% secure. We cannot guarantee absolute security.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">6. Data Retention</h2>
            <p className="text-muted-foreground leading-relaxed mb-4">
              We retain your information for as long as your account is active or as needed to provide services. Specifically:
            </p>
            <ul className="list-disc list-inside text-muted-foreground space-y-2">
              <li><strong>Account Data:</strong> Retained until account deletion</li>
              <li><strong>Usage Logs:</strong> Retained for 90 days</li>
              <li><strong>Pod Session Data:</strong> Automatically deleted upon pod termination or after 30 days of inactivity</li>
              <li><strong>Billing Records:</strong> Retained for 7 years as required by law</li>
            </ul>
            <p className="text-muted-foreground leading-relaxed mt-4">
              You may request deletion of your data at any time by contacting us or deleting your account.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">7. Your Rights and Choices</h2>
            <p className="text-muted-foreground leading-relaxed mb-4">
              Depending on your location, you may have the following rights:
            </p>
            <ul className="list-disc list-inside text-muted-foreground space-y-2">
              <li><strong>Access:</strong> Request a copy of your personal information</li>
              <li><strong>Correction:</strong> Request correction of inaccurate information</li>
              <li><strong>Deletion:</strong> Request deletion of your personal information</li>
              <li><strong>Portability:</strong> Request your data in a portable format</li>
              <li><strong>Objection:</strong> Object to certain processing activities</li>
              <li><strong>Restriction:</strong> Request restriction of processing</li>
              <li><strong>Withdraw Consent:</strong> Withdraw consent where processing is based on consent</li>
            </ul>
            <p className="text-muted-foreground leading-relaxed mt-4">
              To exercise these rights, contact us at{" "}
              <a href="mailto:privacy@agentsmesh.ai" className="text-primary hover:underline">
                privacy@agentsmesh.ai
              </a>
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">8. International Data Transfers</h2>
            <p className="text-muted-foreground leading-relaxed">
              Your information may be transferred to and processed in countries other than your country of residence. We ensure appropriate safeguards are in place for such transfers, including Standard Contractual Clauses approved by relevant authorities.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">9. Cookies and Tracking Technologies</h2>
            <p className="text-muted-foreground leading-relaxed mb-4">
              We use cookies and similar technologies to:
            </p>
            <ul className="list-disc list-inside text-muted-foreground space-y-2">
              <li><strong>Essential Cookies:</strong> Required for authentication and security</li>
              <li><strong>Preference Cookies:</strong> Remember your settings and preferences</li>
              <li><strong>Analytics Cookies:</strong> Help us understand how you use the Service</li>
            </ul>
            <p className="text-muted-foreground leading-relaxed mt-4">
              You can manage cookie preferences through your browser settings. Note that disabling certain cookies may impact Service functionality.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">10. Children&apos;s Privacy</h2>
            <p className="text-muted-foreground leading-relaxed">
              The Service is not intended for users under 16 years of age. We do not knowingly collect personal information from children under 16. If we discover that we have collected information from a child under 16, we will promptly delete it.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">11. Third-Party Links</h2>
            <p className="text-muted-foreground leading-relaxed">
              The Service may contain links to third-party websites or services. We are not responsible for the privacy practices of these third parties. We encourage you to review their privacy policies.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">12. Changes to This Policy</h2>
            <p className="text-muted-foreground leading-relaxed">
              We may update this Privacy Policy from time to time. We will notify you of material changes by posting the new policy on this page and updating the &quot;Last updated&quot; date. For significant changes, we will provide additional notice, such as an email notification.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">13. Contact Us</h2>
            <p className="text-muted-foreground leading-relaxed">
              If you have questions about this Privacy Policy or our data practices, please contact us at:
            </p>
            <div className="mt-4 p-4 bg-muted rounded-lg text-muted-foreground">
              <p><strong>AgentsMesh, Inc.</strong></p>
              <p>Email:{" "}
                <a href="mailto:privacy@agentsmesh.ai" className="text-primary hover:underline">
                  privacy@agentsmesh.ai
                </a>
              </p>
            </div>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">14. California Privacy Rights</h2>
            <p className="text-muted-foreground leading-relaxed mb-4">
              California residents have additional rights under the California Consumer Privacy Act (CCPA):
            </p>
            <ul className="list-disc list-inside text-muted-foreground space-y-2">
              <li>Right to know what personal information is collected</li>
              <li>Right to know whether personal information is sold or disclosed</li>
              <li>Right to opt-out of the sale of personal information (we do not sell personal information)</li>
              <li>Right to non-discrimination for exercising privacy rights</li>
            </ul>
          </section>

          <section>
            <h2 className="text-2xl font-semibold mb-4">15. European Privacy Rights (GDPR)</h2>
            <p className="text-muted-foreground leading-relaxed mb-4">
              If you are located in the European Economic Area (EEA), you have rights under the General Data Protection Regulation (GDPR). Our legal bases for processing include:
            </p>
            <ul className="list-disc list-inside text-muted-foreground space-y-2">
              <li><strong>Contract:</strong> Processing necessary to perform our contract with you</li>
              <li><strong>Legitimate Interests:</strong> Processing for our legitimate business interests</li>
              <li><strong>Consent:</strong> Processing based on your consent</li>
              <li><strong>Legal Obligation:</strong> Processing necessary to comply with laws</li>
            </ul>
            <p className="text-muted-foreground leading-relaxed mt-4">
              You may lodge a complaint with your local data protection authority if you believe we have violated your privacy rights.
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
