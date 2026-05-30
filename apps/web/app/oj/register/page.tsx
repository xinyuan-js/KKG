import { redirect } from "next/navigation";

export default function OJRegisterPage() {
  redirect("/register?redirect=/oj");
}
