import { redirect } from "next/navigation";

export default function OJLoginPage() {
  redirect("/login?redirect=/oj");
}
