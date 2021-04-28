package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"path"
	"strings"
	"sync"
	"time"
	"week7/service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

//расширение типа из сгенерированного файлом
type Biz struct {
	service.UnimplementedBizServer
}

type Admin struct {
	service.UnimplementedAdminServer
	uMan *UserManager
}

//менеджер подключенных пользователей
type UserManager struct {
	id    int
	users map[int]chan *service.Event
	mu    *sync.Mutex
	ACL   map[string][]string
}

func NewUserManager() *UserManager {
	return &UserManager{
		users: make(map[int]chan *service.Event),
		mu:    &sync.Mutex{},
	}
}

//отправляет новое событие во все каналы
func (uMan *UserManager) Mailing(event *service.Event) {
	uMan.mu.Lock()
	defer uMan.mu.Unlock()

	for _, ch := range uMan.users {
		ch <- event
	}
}

func (uMan *UserManager) NewUser() (int, chan *service.Event) {
	uMan.mu.Lock()
	defer uMan.mu.Unlock()

	uMan.id++
	uMan.users[uMan.id] = make(chan *service.Event)
	fmt.Println("USER ", uMan.id, " ADDED")
	return uMan.id, uMan.users[uMan.id]
}

func (uMan *UserManager) DeleteUser(id int) {
	uMan.mu.Lock()
	defer uMan.mu.Unlock()
	if user, ok := uMan.users[id]; ok {
		close(user)
		delete(uMan.users, id)
		fmt.Println("USER ", id, " DELETED")
	}
}

func (uMan *UserManager) DeleteAllUsers() {
	for id, _ := range uMan.users {
		uMan.DeleteUser(id)
	}
}

func NewBiz() *Biz {
	return &Biz{}
}

func NewAdmin(uman *UserManager) *Admin {
	return &Admin{
		uMan: uman,
	}
}

func (b *Biz) Check(ctx context.Context, in *service.Nothing) (*service.Nothing, error) {
	return &service.Nothing{Dummy: true}, nil
}

func (b *Biz) Add(ctx context.Context, in *service.Nothing) (*service.Nothing, error) {
	return &service.Nothing{Dummy: true}, nil
}

func (b *Biz) Test(ctx context.Context, in *service.Nothing) (*service.Nothing, error) {
	return &service.Nothing{Dummy: true}, nil
}

//логирует вызываемые методы
func (a *Admin) Logging(req *service.Nothing, srv service.Admin_LoggingServer) error {
	id, ch := a.uMan.NewUser()
	defer a.uMan.DeleteUser(id)
	for event := range ch {
		fmt.Println("->", event)
		srv.Send(event)

	}
	return nil
}

//счетчик по вызываемым методам
func (a *Admin) Statistics(i *service.StatInterval, srv service.Admin_StatisticsServer) error {
	id, ch := a.uMan.NewUser()
	defer a.uMan.DeleteUser(id)
	stat := service.Stat{
		ByMethod:   make(map[string]uint64),
		ByConsumer: make(map[string]uint64),
	}

	t := time.NewTicker(time.Duration(i.IntervalSeconds) * time.Second)
	defer t.Stop()
	for {
		select {
		case event, ok := <-ch:
			if ok {
				stat.ByConsumer[event.Consumer] += 1
				stat.ByMethod[event.Method] += 1
			} else {
				return nil
			}
		case <-t.C:
			stat.Timestamp = time.Now().Unix()
			if err := srv.Send(&stat); err != nil {
				return err
			}
			stat = service.Stat{
				ByMethod:   make(map[string]uint64),
				ByConsumer: make(map[string]uint64),
			}
		}
	}
}

func (uMan *UserManager) CheckMethod(consumer string, method string) bool {
	for _, m := range uMan.ACL[consumer] {
		_, base := path.Split(m)
		if method == m || base == "*" {
			return true
		}
	}
	return false
}

func (uMan *UserManager) Interceptor(ctx context.Context, method string) error {
	md, _ := metadata.FromIncomingContext(ctx)
	host := ""
	consumer := strings.Join(md.Get("consumer"), "")

	if p, ok := peer.FromContext(ctx); ok {
		host = p.Addr.String()
	}

	uMan.Mailing(&service.Event{
		Timestamp: time.Now().Unix(), // number of seconds since January 1, 1970 UTC
		Consumer:  consumer,
		Method:    method,
		Host:      host,
	})

	if !uMan.CheckMethod(consumer, method) {
		fmt.Println("acces denied")
		return status.Errorf(codes.Unauthenticated, "access denied")
	}

	return nil
}

func (uMan *UserManager) authInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if err := uMan.Interceptor(ctx, info.FullMethod); err != nil {
		return nil, err
	}
	return handler(ctx, req)
}

func (uMan *UserManager) streamInterceptor(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	if err := uMan.Interceptor(stream.Context(), info.FullMethod); err != nil {
		return err
	}

	return handler(srv, stream)
}

//&{map[biz_admin:[/main.Biz/*] biz_user:[/main.Biz/Check /main.Biz/Add] logger:[/main.Admin/Logging] stat:[/main.Admin/Statistics]]} <nil>
func ParseACL(ACLData string) (map[string][]string, error) {
	data := make(map[string][]string)
	err := json.Unmarshal([]byte(ACLData), &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func StartMyMicroservice(ctx context.Context, listenAddr string, ACLData string) error {
	acl, err := ParseACL(ACLData)
	if err != nil {
		fmt.Println("ParseACLData error: ", err)
		return err
	}
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalln("can't listen address", err)
		return err
	}
	uMan := NewUserManager()
	uMan.ACL = acl
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(uMan.authInterceptor),
		grpc.StreamInterceptor(uMan.streamInterceptor),
	)
	service.RegisterBizServer(grpcServer, NewBiz())
	service.RegisterAdminServer(grpcServer, NewAdmin(uMan))
	go grpcServer.Serve(listener)
	go func() {
		<-ctx.Done()
		// time.Sleep(time.Duration(1) * time.Second)
		uMan.DeleteAllUsers()
		grpcServer.GracefulStop()
	}()
	return nil
}
