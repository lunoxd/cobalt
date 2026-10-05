import * as React from "react"
import { cn } from "@/lib/utils"

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: "default" | "destructive" | "outline" | "secondary" | "ghost" | "link"
  size?: "default" | "sm" | "lg" | "icon"
}

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant = "default", size = "default", ...props }, ref) => {
    const variants = {
      default: "bg-blue-600 text-white shadow hover:bg-blue-500 active:bg-blue-700",
      destructive: "bg-red-600 text-white shadow-sm hover:bg-red-500",
      outline: "border border-zinc-700 bg-transparent shadow-sm hover:bg-zinc-800 text-zinc-200",
      secondary: "bg-zinc-800 text-zinc-100 shadow-sm hover:bg-zinc-700",
      ghost: "hover:bg-zinc-850 hover:text-zinc-50 text-zinc-400 hover:text-zinc-200",
      link: "text-blue-400 underline-offset-4 hover:underline",
    }

    const sizes = {
      default: "h-9 px-4 py-2 text-sm",
      sm: "h-8 rounded-md px-3 text-xs",
      lg: "h-10 rounded-md px-8 text-base",
      icon: "h-9 w-9",
    }

    return (
      <button
        ref={ref}
        className={cn(
          "inline-flex items-center justify-center whitespace-nowrap rounded-md font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-zinc-400 disabled:pointer-events-none disabled:opacity-50 cursor-pointer",
          variants[variant],
          sizes[size],
          className
        )}
        {...props}
      />
    )
  }
)
Button.displayName = "Button"
