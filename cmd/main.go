package cmd

func main () {
	panic("Empty")	
/*  
	app := NewApp()

    go func() {
        if err := app.Run(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("server error: %v", err)
        }
    }()

	stop := make(chan os.Signal, 1)
    signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

    <-stop

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := app.Shutdown(ctx); err != nil {
        log.Fatalf("shutdown failed: %v", err)
    } */
}