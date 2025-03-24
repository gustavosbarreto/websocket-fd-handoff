local ffi = require "ffi"
local C = ffi.C

ffi.cdef [[
    typedef unsigned int socklen_t;
    typedef unsigned long size_t;
    typedef long ssize_t;

    struct cmsghdr {
        size_t cmsg_len;
        int cmsg_level;
        int cmsg_type;
    };

    struct iovec {
        void *iov_base;
        size_t iov_len;
    };

    struct msghdr {
        void *msg_name;
        socklen_t msg_namelen;
        struct iovec *msg_iov;
        size_t msg_iovlen;
        void *msg_control;
        size_t msg_controllen;
        int msg_flags;
    };

    struct sockaddr_un {
        unsigned short sun_family;
        char sun_path[108];
    };

    int socket(int domain, int type, int protocol);
    int connect(int sockfd, const struct sockaddr *addr, socklen_t addrlen);
    ssize_t sendmsg(int sockfd, const struct msghdr *msg, int flags);
    int close(int fd);
    int dup2(int oldfd, int newfd);
    const char *strerror(int errnum);

    enum {
        AF_UNIX = 1,
        AF_INET = 2,
        SOCK_STREAM = 1,
        SOCK_DGRAM = 2,
        SOL_SOCKET = 1,
        SCM_RIGHTS = 1
    };

    typedef int (*ngx_http_lua_socket_tcp_retval_handler_masked)(void *r, void *u, void *L);
    typedef void (*ngx_http_lua_socket_tcp_upstream_handler_pt_masked)(void *r, void *u);

    typedef struct {
        void *data;
        void *read;
        void *write;
        int   fd;
    } ngx_connection_s;

    typedef struct {
        ngx_connection_s *connection;
    } ngx_peer_connection_s;

    typedef struct {
        ngx_http_lua_socket_tcp_retval_handler_masked read_prepare_retvals;
        ngx_http_lua_socket_tcp_retval_handler_masked write_prepare_retvals;
        ngx_http_lua_socket_tcp_upstream_handler_pt_masked read_event_handler;
        ngx_http_lua_socket_tcp_upstream_handler_pt_masked write_event_handler;

        void *udata_queue;
        void *socket_pool;
        void *conf;
        void *cleanup;
        void *request;

        ngx_peer_connection_s peer;
    } ngx_http_lua_socket_tcp_upstream_s;

    typedef struct ngx_http_lua_socket_tcp_upstream_s ngx_http_lua_socket_tcp_upstream_t;
]]

return ffi
