import { useState } from "react";
import { Outlet } from "react-router-dom";
import { Sidebar } from "./sidebar";
import { Topbar, DesktopTopbar } from "./topbar";
import { MobileNav } from "./mobile-nav";
import { CommandPalette } from "./command-palette";
import { useIsMobile } from "@/hooks/use-media-query";

export function RootLayout() {
  const [mobileNavOpen, setMobileNavOpen] = useState(false);
  const isMobile = useIsMobile();

  return (
    <div className="flex h-screen overflow-hidden bg-[#131313]">
      <CommandPalette />

      {/* Sidebar - Desktop only */}
      <Sidebar />

      {/* Mobile Navigation Sheet */}
      {isMobile && (
        <MobileNav open={mobileNavOpen} onOpenChange={setMobileNavOpen} />
      )}

      {/* Main Content Area */}
      <div className="flex flex-1 flex-col overflow-hidden ml-0 md:ml-64">
        {/* Topbar - Mobile only (hamburger) / Desktop (search + profile) */}
        {isMobile ? (
          <Topbar onMobileMenuToggle={() => setMobileNavOpen(true)} />
        ) : (
          <DesktopTopbar />
        )}

        {/* Page Content */}
        <main className="flex-1 overflow-y-auto p-4 md:p-6 lg:p-10">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
