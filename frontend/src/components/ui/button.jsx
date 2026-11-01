import * as React from "react"
import { Slot } from "@radix-ui/react-slot"
import { cn } from "@/lib/utils"

const variantClasses = {
  default: "bg-primary text-primary-foreground hover:bg-primary/90 shadow-sm",
  destructive: "bg-destructive text-destructive-foreground hover:bg-destructive/90 shadow-sm",
  outline: "border border-input bg-background hover:bg-accent hover:text-accent-foreground shadow-sm",
  secondary: "bg-secondary text-secondary-foreground hover:bg-secondary/80 shadow-sm",
  ghost: "hover:bg-accent hover:text-accent-foreground",
  link: "text-primary underline-offset-4 hover:underline",
}

const sizeClasses = {
  default: "h-9 px-4 py-2 text-sm",
  sm: "h-8 rounded-md px-3 text-xs",
  lg: "h-10 rounded-md px-8 text-sm",
  icon: "h-9 w-9",
  "icon-sm": "h-8 w-8",
  "icon-xs": "h-6 w-6",
  "icon-lg": "h-10 w-10",
  xs: "h-6 rounded-md px-2 text-xs",
}

const Button = React.forwardRef(
  ({ className, variant = "default", size = "default", asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : "button"
    return (
      <Comp
        ref={ref}
        className={cn(
          "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0",
          variantClasses[variant] || variantClasses.default,
          sizeClasses[size] || sizeClasses.default,
          className
        )}
        {...props}
      />
    )
  }
)
Button.displayName = "Button"

export { Button }

