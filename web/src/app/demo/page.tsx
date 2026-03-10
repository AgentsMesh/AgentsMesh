"use client";

import { useState } from "react";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Logo } from "@/components/common";
import { useTranslations } from "next-intl";

export default function DemoRequestPage() {
  const t = useTranslations();
  const [formData, setFormData] = useState({
    name: "",
    email: "",
    company: "",
    referral: "",
    message: "",
  });
  const [loading, setLoading] = useState(false);
  const [submitted, setSubmitted] = useState(false);
  const [error, setError] = useState("");

  const handleChange = (
    e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>,
  ) => {
    setFormData((prev) => ({
      ...prev,
      [e.target.name]: e.target.value,
    }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError("");

    try {
      // Send as mailto fallback — replace with API endpoint when available
      const subject = encodeURIComponent(`Demo Request from ${formData.name} (${formData.company})`);
      const body = encodeURIComponent(
        `Name: ${formData.name}\nEmail: ${formData.email}\nCompany: ${formData.company}\nHow did you hear about us: ${formData.referral}\n\nMessage:\n${formData.message}`,
      );
      window.location.href = `mailto:bd@agentsmesh.ai?subject=${subject}&body=${body}`;
      setSubmitted(true);
    } catch {
      setError(t("landing.demo.error"));
    } finally {
      setLoading(false);
    }
  };

  if (submitted) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background px-4">
        <div className="w-full max-w-md text-center space-y-6">
          <div className="w-16 h-16 rounded-full bg-green-500/10 flex items-center justify-center mx-auto">
            <svg className="w-8 h-8 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
            </svg>
          </div>
          <h1 className="text-2xl font-bold">{t("landing.demo.thankYou")}</h1>
          <p className="text-muted-foreground">{t("landing.demo.thankYouDescription")}</p>
          <Link href="/">
            <Button variant="outline">{t("landing.demo.backToHome")}</Button>
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4 py-8">
      <div className="w-full max-w-md space-y-6">
        {/* Header */}
        <div className="text-center">
          <Link href="/" className="inline-flex items-center gap-2">
            <div className="w-10 h-10 rounded-lg overflow-hidden">
              <Logo />
            </div>
            <span className="text-2xl font-bold text-foreground">AgentsMesh</span>
          </Link>
          <h1 className="mt-6 text-2xl font-semibold text-foreground">
            {t("landing.demo.title")}
          </h1>
          <p className="mt-2 text-sm text-muted-foreground">
            {t("landing.demo.subtitle")}
          </p>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} className="space-y-4">
          {error && (
            <div className="p-3 text-sm text-destructive bg-destructive/10 rounded-md">
              {error}
            </div>
          )}

          <div className="space-y-2">
            <label htmlFor="name" className="text-sm font-medium text-foreground">
              {t("landing.demo.nameLabel")}
            </label>
            <Input
              id="name"
              name="name"
              type="text"
              placeholder={t("landing.demo.namePlaceholder")}
              value={formData.name}
              onChange={handleChange}
              required
            />
          </div>

          <div className="space-y-2">
            <label htmlFor="email" className="text-sm font-medium text-foreground">
              {t("landing.demo.emailLabel")}
            </label>
            <Input
              id="email"
              name="email"
              type="email"
              placeholder={t("landing.demo.emailPlaceholder")}
              value={formData.email}
              onChange={handleChange}
              required
            />
          </div>

          <div className="space-y-2">
            <label htmlFor="company" className="text-sm font-medium text-foreground">
              {t("landing.demo.companyLabel")}
            </label>
            <Input
              id="company"
              name="company"
              type="text"
              placeholder={t("landing.demo.companyPlaceholder")}
              value={formData.company}
              onChange={handleChange}
              required
            />
          </div>

          <div className="space-y-2">
            <label htmlFor="referral" className="text-sm font-medium text-foreground">
              {t("landing.demo.referralLabel")}
            </label>
            <Input
              id="referral"
              name="referral"
              type="text"
              placeholder={t("landing.demo.referralPlaceholder")}
              value={formData.referral}
              onChange={handleChange}
            />
          </div>

          <div className="space-y-2">
            <label htmlFor="message" className="text-sm font-medium text-foreground">
              {t("landing.demo.messageLabel")}
            </label>
            <textarea
              id="message"
              name="message"
              placeholder={t("landing.demo.messagePlaceholder")}
              value={formData.message}
              onChange={handleChange}
              rows={4}
              className="flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 resize-none"
            />
          </div>

          <Button type="submit" className="w-full" disabled={loading}>
            {loading ? t("landing.demo.submitting") : t("landing.demo.submit")}
          </Button>
        </form>

        <p className="text-center text-sm text-muted-foreground">
          <Link href="/" className="text-primary hover:underline">
            {t("landing.demo.backToHome")}
          </Link>
        </p>
      </div>
    </div>
  );
}
