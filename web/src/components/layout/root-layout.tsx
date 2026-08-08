import { useEffect, useRef, useState } from "react";
import { Outlet, useLocation } from "react-router-dom";
import { Sidebar } from "./sidebar";
import { Topbar, DesktopTopbar } from "./topbar";
import { MobileNav } from "./mobile-nav";
import { CommandPalette } from "./command-palette";
import { useIsMobile } from "@/hooks/use-media-query";

export function RootLayout() {
  const [mobileNavOpen, setMobileNavOpen] = useState(false);
  const isMobile = useIsMobile();
  const location = useLocation();
  const mainRef = useRef<HTMLElement>(null);

  useEffect(() => {
    mainRef.current?.scrollTo({ top: 0, behavior: "instant" });
  }, [location.pathname, location.search]);

  return (
    <div className="flex h-screen overflow-hidden bg-background">
      <CommandPalette />

      {/* Sidebar - Desktop only */}
      <Sidebar />

      {/* Mobile Navigation Sheet */}
      {isMobile && (
        <MobileNav open={mobileNavOpen} onOpenChange={setMobileNavOpen} />
      )}

      {/* Main Content Area */}
      <div className="ml-0 flex flex-1 flex-col overflow-hidden md:ml-60">
        {/* Topbar - Mobile only (hamburger) / Desktop (search + profile) */}
        {isMobile ? (
          <Topbar onMobileMenuToggle={() => setMobileNavOpen(true)} />
        ) : (
          <DesktopTopbar />
        )}

        {/* Page Content */}
        <main ref={mainRef} className="flex-1 overflow-y-auto">
          <div className="mx-auto w-full max-w-[1248px] p-4 md:p-6 lg:p-8">
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  );
}
