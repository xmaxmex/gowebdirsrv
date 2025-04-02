package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

type myService struct{}

func (m *myService) Execute(args []string, req <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	status <- svc.Status{State: svc.StartPending}
	go func() {

		IniciarHTTPDir()
		//time.Sleep(1 * time.Minute)

	}()
	status <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

loop:
	for {
		select {
		case c := <-req:
			switch c.Cmd {
			case svc.Stop, svc.Shutdown:
				status <- svc.Status{State: svc.StopPending}
				break loop
			default:
				continue
			}
		}
	}
	status <- svc.Status{State: svc.Stopped}
	return false, 0
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "install":
			exepath, err := os.Executable()
			if err != nil {
				log.Fatalf("Failed to get executable path: %v", err)
			}

			err = installService2("vmshttpdir", exepath, "Serviço que server http os diretorios de records do Multi VMS")
			if err != nil {
				log.Fatalf("Failed to install service: %v", err)
			}
			log.Print("Service installed successfully")
			return
		case "remove":
			err := removeService("vmshttpdir")
			if err != nil {
				log.Fatalf("Failed to remove service: %v", err)
			}
			log.Print("Service removed successfully")
			return
		}
	}

	isInteractive, err := svc.IsAnInteractiveSession()
	if err != nil {
		log.Fatalf("Failed to determine if we are running in an interactive session: %v", err)
	}

	if isInteractive {
		runInteractive()
	} else {
		runService("vmshttpdir", false)
	}
}

func runService(name string, isDebug bool) {
	err := svc.Run(name, &myService{})
	if err != nil {
		log.Fatalf("%s service failed: %v", name, err)
	}
}

func runInteractive() {
	log.Print("Running in interactive mode")
	go func() {

		IniciarHTTPDir()
		//time.Sleep(1 * time.Minute)

	}()
	select {}
}

func installService2(name, exepath, desc string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err == nil {
		s.Close()
		return fmt.Errorf("serviço %s já existe", name)
	}

	s, err = m.CreateService(name, exepath, mgr.Config{
		StartType:   mgr.StartAutomatic,
		Description: desc, // Adiciona a descrição aqui
	})
	if err != nil {
		return err
	}
	defer s.Close()

	err = eventlog.InstallAsEventCreate(name, eventlog.Error|eventlog.Warning|eventlog.Info)
	if err != nil {
		s.Delete()
		return err
	}
	return nil
}

func removeService(name string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err != nil {
		return nil
	}
	defer s.Close()

	err = s.Delete()
	if err != nil {
		return err
	}

	err = eventlog.Remove(name)
	if err != nil {
		return err
	}

	return nil
}

func IniciarHTTPDir() {

	dir, err := os.Getwd()
	fmt.Println("Diretorio atual antes do Chdir: ", dir)

	err = os.Chdir("c:\\multivms\\records")
	if err != nil {
		fmt.Sprintf("Falha ao definir o diretório de trabalho: %v", err)
		return
	}

	dir, err = os.Getwd()
	log.Println("Diretorio atual depois do Chdir: ", dir)

	fs := http.FileServer(http.Dir(dir))
	http.Handle("/", fs)
	log.Println("Servidor HTTP iniciado na porta 8001")
	err = http.ListenAndServe(":8001", nil)
	if err != nil {
		fmt.Sprintf("Falha ao iniciar o servidor HTTP: %v", err)
	}

}
