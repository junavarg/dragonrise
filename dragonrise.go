// dragonrise
//
// Devuelve por stdout y publica en un MQTT broker eventos y estado de interruptores y ejes(conmutadores) 
// de la tarjeta 'Generic USB joystick' DragonRise para uso en IoT, home automation, robotics, ...
// Autor: junavarg

package main  

import (
	"fmt"
    "os"
	"flag"
	"encoding/binary"
	"time" 
	"path/filepath"
	"io/ioutil"
	"net"
	"net/url"
	"encoding/json"
	"github.com/eclipse/paho.mqtt.golang"
	"crypto/tls"
	"crypto/md5"
)
// constantes
const(
	versionFecha = "v031 - 16 mayo 2020"  // inclusion de device en JSON goroutine leerDevice(device) 
	bufferSize =8 //numero de bytes de buffer de lectura
	nombreFicheroDispositivoOmision = "/dev/input/js0" 
	statusFilePath = "/var/lib/dragonrise/"   
	statusFileExt= ".dat"
	maxSwt = 12    //número máximo de interruptores
	maxCom = 7     //número máximo de ejes-conmutadores 
	maxTarjetas =10 //número maximo de tarjetas (files devices) que se almacenaran en un array
)

// constantes de mqtt
const (
	maxNumberOfBroquers = 6
	sufijoFinalTopic="event"
	lwtMensaje="{\"time\":0,\"event\":{\"type\":-1,\"sensor\":0,\"value\":0}}" // mensaje Last Will que emitirá el broker cuando el publicador se desconecta inexperadamente
)

//definicion de tipos
type evento struct{
	TipoSensor byte 	`json:"type"`
	NumSensor byte 	    `json:"sensor"`
	ValorSensor int16	`json:"value"`
}

type dragonrise struct{
	Tiempo int64 		`json:"time"`
	Dispositivo string	`json:"device"`
	Evento evento   	`json:"event"`// último evento registrado
	Swt []int16   		`json:"switches"`// estado actual de interruptores
	Com []int16   		`json:"axes"`// estado actual de ejes/conmutadores
}

//variables globales
var (
	switches [maxTarjetas][maxSwt]int16     // array de interruptores
	conmutadores [maxTarjetas][maxCom]int16 // array de ejes/conmutadores
	tarjeta [maxTarjetas]dragonrise			// array de estructuras de ultimo evento y estados	
	
	fDispositivo [maxTarjetas]string 	//nombre de fichero de dispositivo
	hDispositivo [maxTarjetas]*os.File 	//handle de fichero de dispositivo
	fEstado [maxTarjetas]string 		//nombre de fichero de estado
	hEstado [maxTarjetas]*os.File 		//handle de fichero de estado
	numTarjetas int						//numero de tarjetas gestionadas por el proceso dragonrise. Siempre numTarjetas <= maxTarjetas
)

//variables globales de MQTT
var (
	broker = [maxNumberOfBroquers]string{ 
	//	"tcp://broker.hivemq.com:1883",
	//	"ws://test.mosquitto.org:8080", 
	//	"tcp://mqtt.eclipse.org:1883",
	//	"tcp://broker.emqx.io:1883",
	//	"wss://mqttws.vigilanet.com:443"
	}
		
	cliente  [len(broker)]mqtt.Client
	opcionesCliente [len(broker)]*mqtt.ClientOptions
	// esta condición es para todos los clientes
	verificarCertificadoBroker = false
	numClientes  int
	topic [maxTarjetas]string  // habra tantos topics como tarjetas/devices Se ajusta el array al máximo
)

func onConnectHandler(c mqtt.Client){
	lectorOpcionesCliente:=c.OptionsReader()
	fmt.Fprintf(os.Stderr,"\nConectado a un servidor: ")
	for _, v:= range lectorOpcionesCliente.Servers(){
		fmt.Fprintf(os.Stderr,"%s",v) 
		fmt.Fprintf(os.Stderr," ")
	}
	fmt.Fprintf(os.Stderr," con clientID %s", lectorOpcionesCliente.ClientID())
	//TODO Descubrir a que server se ha conectado si se ha metido más de uno con AddBroker
}

func onConnetionLostHandler(c mqtt.Client, er error ){
	lectorOpcionesCliente:=c.OptionsReader()
	fmt.Fprintf(os.Stderr,"\nconexión perdida. err: ",lectorOpcionesCliente.Servers()[0] , er)
}

func onReconnectingHandler(c mqtt.Client, co *mqtt.ClientOptions){
	fmt.Fprintf(os.Stderr,"\n¡¡ inicio reconexión ... !! ")
}

// Conecta con broker con protocolo y puyerto indicado el la url. 
// En la URL se puede especificar el usuario y contraseña
// TODO: Permite especificar varias URLs para diferentes servidores y protocolos para HA pero usuario contraseña han de ser el mismo. 
func inicioConexion(urlBroker ...string){
	if numClientes>=len(broker){
		return 
	} 
	numCliente:=numClientes
	opcionesCliente[numCliente] = mqtt.NewClientOptions()

	urlMqttBroker:=""
	usuario:=string("")
	password:=string("") 	
	for _, v:= range urlBroker{
		//se aisla usuario/password de la url
		uri, _ := url.Parse(v)
		urlMqttBroker = fmt.Sprintf("%s://%s", uri.Scheme, uri.Host)
		opcionesCliente[numCliente].AddBroker(urlMqttBroker)
		//TODO controlar que si se pasan 2 o mas URL el usuario y password coinciden
		pUsuariopassword:=uri.User
		if pUsuariopassword != nil {
			usuario=(*pUsuariopassword).Username()
			password, _=(*pUsuariopassword).Password()
		}
	}

	//Obtención de un clientID para cliente MQQT activo
	// No pueden conectarse al broker dos cliente con mismo ClientID
	var preClientID string // clientID antes de hacer el hash
	var clientID string	 // hash-MD5 de preClientID

	// se obtienen todas las MACs del dispositivo
	macs, _:=getMacAddr() 
	// Se pone como preClientID la primera MAC - seguido del fichero de dispositivo - seguido numero de cliente. 
	preClientID = (macs[0] + filepath.Base(flag.Arg(0)) + string(numCliente)) 	
	// clientID es el hash-MD5 de preClientID truncado a 10caracterews hexadecimales
	clientID = fmt.Sprintf("%x", md5.Sum([]byte(preClientID)))[0:10]
	
	//TODO retirar o controlar con opcion en linea de comando
	//Si se pasa un clinteID con cero caracteres el broker le asigna uno aleatorio interno que no comunica al propio cliente
	//Esto puede tener limitaciones.  
	//clientID=""
	opcionesCliente[numCliente].
		SetUsername(usuario).
		SetPassword(password).	
		SetClientID(clientID).
		SetConnectTimeout(10 * time.Second).
		SetConnectRetry(true).   //importante a TRUE en sistemas que deben estar siempre conectados, incluso en rearranque
		SetConnectRetryInterval(30 * time.Second).
		SetKeepAlive(30 * time.Second).
		SetPingTimeout(5 * time.Second).
		//TODO arreglar lwt topic no puede llevar wildcards 
		SetWill(topic[0],lwtMensaje, 0, true).
		SetOnConnectHandler(onConnectHandler).	
		SetConnectionLostHandler(onConnetionLostHandler).
		SetReconnectingHandler(onReconnectingHandler)

		// No verifica certificado de l broker en tls y wss
		// Salvo que se especifique opcion -cbc, se ajusta la configuracion TLS para que no se verifique el certificado que presente el broker
		if verificarCertificadoBroker == false {
			opcionesCliente[numCliente].SetTLSConfig(&tls.Config{InsecureSkipVerify: true})
		}

	cliente[numCliente] = mqtt.NewClient(opcionesCliente[numCliente])
	numClientes++
	//conexion inicial asíncrona una vez establecidos los clientes mediante función anónima
	go func (numCliente int){
		fmt.Fprintf(os.Stderr,"\nConexión inicial ...")
		if token := cliente[numCliente].Connect(); token.Wait() && token.Error() != nil {
			fmt.Fprintf(os.Stderr,"\nError de conexión inicial:", token.Error())  //Nunca pasa por aquí si ConnectRetry en opciones de cliente esta a True 
		} 
	}(numCliente)
}



func publicar(topic string, carga string){
	for i:=0; i<numClientes; i++{
		if cliente[i].IsConnectionOpen(){
			cliente[i].Publish(topic, 0, true, carga) // se publica con qos=0 y retention=true
			fmt.Fprintf(os.Stderr," ok->%d",i)
		} else{
			fmt.Fprintf(os.Stderr," ko->%d",i)
		}
	}
}

/*
devuelve topic completo de publicacion formado por los siguientes elementos separados por "/"
	baseTopic que está el la url que se pasa en mqpub, 
	el fichero de dispositivo, que se pasa en device
	el sufijo final 
*/	
func devuelveTopic(mqpub string, device string)(topic string) {
	uri, _ := url.Parse(mqpub)
	baseTopic:=filepath.Base(uri.Path)
	deviceFile:=filepath.Base(device)
	topic=fmt.Sprintf("%s/%s/%s", baseTopic, deviceFile, sufijoFinalTopic)
	return topic
}

func getMacAddr() ([]string, error) {
    ifas, err := net.Interfaces()
    if err != nil {
        return nil, err
    }
	// fmt.Println(ifas)   //para depuracion
	// llena un array de strings con cada interfaz que tenga mac address
    var as []string
    for _, ifa := range ifas {
        a := ifa.HardwareAddr.String()
        if a != "" {
            as = append(as, a)
        }
    }
    return as, nil
}

/*  
//Registra eventos sinteticos (de conocimiento estado inicial) y reales 
//en la struct de estado dragonrise, salvo que tipoSensor == 0
//Saca por stdout los valores de la struct, salvo eventoReal==false 
*/
func tratarEvento (numTarjeta int, eventoReal bool, tipoSensor byte, numSensor byte, valorSensor int16) (error int){
	switch tipoSensor{
		//no registra estado en struct. Se asegura que numSensor y valorSensor sean 0
		case 0:	
			numSensor=0
			valorSensor=0
		// en estos dos casos SI registra estado en struct
		case 1:
			tarjeta[numTarjeta].Swt[numSensor] = valorSensor
		case 2:
			//Descarta eventos espureos de ejes distitos a 0 o 1 ¡¡cuidadito con el algebra de Boole!!
			if !(numSensor==0 || numSensor==1){
				return 1
			}
			valorSensor=  valorSensor/32767   //si es tipo eje se normaliza el valor (-1 0 +1)
			tarjeta[numTarjeta].Com[numSensor]= valorSensor;
	}
	tarjeta[numTarjeta].Evento.TipoSensor = tipoSensor
	tarjeta[numTarjeta].Evento.NumSensor = numSensor
	tarjeta[numTarjeta].Evento.ValorSensor = valorSensor 
	tarjeta[numTarjeta].Dispositivo = filepath.Base(fDispositivo[numTarjeta])
	tarjeta[numTarjeta].Tiempo = time.Now().Unix()  // --> evento real

	if tipoSensor == 0 || eventoReal == true{
		salida, _ := json.Marshal(&tarjeta[numTarjeta])
		fmt.Printf("\n%s", string(salida))	//Sale por stdout, no por stderr
		//TODO: Tratar errores
		hEstado[numTarjeta].Truncate(0)
		hEstado[numTarjeta].Seek(0,0)
		hEstado[numTarjeta].Write([]byte(salida))
		hEstado[numTarjeta].Sync()
	}
	return 0
}


func reinicializaDragonrise(nDisp int) {
	var err error
	
	device:=fDispositivo[nDisp]
			
	pintadoError1:=false
	pintadoError2:=false
	pintadoError3:=false
	
	fmt.Fprintf(os.Stderr, "\nAbriendo dispositivo %s.", device)
	inicio:
	hDispositivo[nDisp].Close()
	hDispositivo[nDisp]=nil
	for hDispositivo[nDisp]==nil  {
		hDispositivo[nDisp], err = os.Open(device)
		if !pintadoError1==false {
			fmt.Fprintf(os.Stderr, "\nNo se puede abrir dispositivo %s . Reintentado en silencio cada 2s ...", device)
			pintadoError1=true
		}
		_=err //para evitar error de no uso
		time.Sleep(2 * time.Second)  //espera para reintento
	}
	
	hEstado[nDisp].Close()
	hEstado[nDisp]=nil
	fmt.Fprintf(os.Stderr,"\nCreando fichero de estado %s", fEstado[nDisp])
	os.Mkdir(statusFilePath, 0755)
	fmt.Fprintf(os.Stderr, "\nAbriendo fichero de estado de interruptores %s", fEstado[nDisp])	
	hEstado[nDisp], err = os.OpenFile(fEstado[nDisp], os.O_RDWR | os.O_CREATE, 0755) //hEstado[] está declarada a nivel global 
	if err!=nil {
		if !pintadoError2 {
			fmt.Fprintf(os.Stderr, "\nError abriendo fichero de estado de interruptores %s. Reintentando en silencio ...", fEstado[nDisp])	
			pintadoError2=true
		}
		goto inicio
	}

	/*
	El controlador joystick /dev/input/js0 tras la apertura del fichero de dispositivo devuelve en orden "eventos sintéticos" (no reales)
	de cada interruptor o eje/conmutador para informar de su estado actual. En el caso de dragonrise primero genera 
	12 eventos para los 12 interuptores y despues 7 eventos para eventos de los	ejes/conmutadores.
	Tras eso las lecturas al dispositivos estaran bloqueadas en espera de eventos reales.
	*/

	leidos:=0
	buffer:= make([]byte, 8)
    for nSwt:=0 ; nSwt<maxSwt; nSwt++ {
		leidos, err = hDispositivo[nDisp].Read(buffer)
		if (err!=nil || leidos!=8 || buffer[6]!=0x81 ){
			if !pintadoError3 {
				fmt.Fprintf(os.Stderr, "\nError de inicializacion switches")
				pintadoError2=true
			}			
			goto inicio
		}
		tipoSensor:=buffer[6]&(0xFF^0x80)
		posicion := buffer[7]
		valor := int16(binary.LittleEndian.Uint16(buffer[4:6]))
	   	tratarEvento(nDisp, false, tipoSensor, posicion ,valor)
	}
	for nCom:=0 ; nCom<maxCom; nCom++ {
		leidos, err = hDispositivo[nDisp].Read(buffer)
		if (err!=nil || leidos!=8 || buffer[6]!=0x82 ){
			if !pintadoError3 {
				fmt.Fprintf(os.Stderr, "\nError de inicializacion switches")
				pintadoError2=true
			}			
			goto inicio
		}
		tipoSensor:=buffer[6]&(0xFF^0x80)
		posicion := buffer[7]
		valor := int16(binary.LittleEndian.Uint16(buffer[4:6]))
		tratarEvento(nDisp, false, tipoSensor, posicion ,valor)
	}
	
	// terminada la inicializacion. Se solicita la salida del estado inicial evento con eventoreal=false y tipo sensor=0 
	// arreglar control errores tratarEvento()
	er:=tratarEvento(nDisp,false,0,0,0)
	_=er
	fmt.Fprintf(os.Stderr, "\nReinicializada dragonrise en %s", device)
}


func leerDevice(numTarjeta int){
	
	var tipoSensor byte
	var posicion byte
	var valor int16
	buffer:= make([]byte, bufferSize)
	for {
		leidos, _ := hDispositivo[numTarjeta].Read(buffer)
		if leidos!=8{
			//TODO tratar error
			fmt.Fprintf(os.Stderr, "\nError: Lectura de evento con menos de 8bytes")
		} 
		
		tipoSensor=buffer[6]&(0xFF^0x80)
		posicion = buffer[7]
		valor = int16(binary.LittleEndian.Uint16(buffer[4:6]))
	  
		er := tratarEvento(numTarjeta, true, tipoSensor, posicion ,valor)
		_=er
/*			
		if (er==0 && mqpub!=""){
*/
		if (er==0 ){
		//Publicacion de evento desde el propio fichero de estado si se tiene mqpub
			estado, _ := ioutil.ReadFile(fEstado[numTarjeta])
			publicar(topic[numTarjeta], string(estado))
		}
	}
}

func main(){
	fmt.Fprintf(os.Stderr,"stderr: dragonrise %s  autor:junav (junav2@hotmail.com)", versionFecha)

	pOpcionH := flag.Bool("h", false, "Muestra más información de ayuda")
	var mqpub string
	var mqpub2 string
	var mqpub3 string

	flag.StringVar(&mqpub,  "mqpub",  "", "URL MQTT Broker y topic de publicacion de mensajes mqttprotocol://host.dominio.tld:puerto/base_topic")
	flag.StringVar(&mqpub2, "mqpub2", "", "URL 2n MQTT Broker y topic de publicacion de mensajes mqttprotocol://host.dominio.tld:puerto/base_topic")
	flag.StringVar(&mqpub3, "mqpub3", "", "URL 3r MQTT Broker y topic de publicacion de mensajes mqttprotocol://host.dominio.tld:puerto/base_topic")

	pOpcionCbc := flag.Bool("cbc", false, "Check Broker Certificate. Si no se pone esta opción el certificado presentado por el MQTT broker no será comprobado")
	_=pOpcionCbc

	flag.Parse() 
	if (*pOpcionH) {
		fmt.Println("Use: dragonrise [options] [device]")	
		fmt.Println("Devuelve por stdout y publica en un MQTT broker eventos y estado de interruptores y ejes(conmutadores) de la tarjeta 'Generic USB joystick' de DragonRise para uso en IoT")
		fmt.Println("Lee fichero de dispositivo /dev/input/js0 u otro especificado")
		fmt.Println("Considera que en la primera lectura se van a recibir 19 eventos sinténticos (12 interruptores y 7 ejes/comuntadores para anotar internamente el estado actual")
		fmt.Println("Sin embargo la tarjeta más común solo facilita conexiones para 2 ejes (sanwa joystick)")
		fmt.Println("El estado se guarda en un fichero '<device>.dat' en /var/lib/dragonrise/")
		fmt.Println("Opciones:")
		fmt.Println("-mqpub <url>")
		fmt.Println("       Especifica la URL de brocker MQTT y la raiz de un topic (basetopic) donde publicar el estado cada vez que se produzca un evento")
		fmt.Println("       Formato URL:    protocolo://[usuario[:password]@]host.dominio.tld:puerto/base_topic")
		fmt.Println("       Ejemplos:")
		fmt.Println("       -mqpub=tcp://host.dominio.tld:1883/base_topic")
		fmt.Println("       -mqpub=ssl://[usuario[:password]@]host.dominio.tld:8883/base_topic")
		fmt.Println("       -mqpub=ws://host.dominio.tld:80/base_topic")
		fmt.Println("       -mqpub=wss://[usuario[:password]@]host.dominio.tld:443/base_topic")
		fmt.Println("       Para enviar credenciales de autenticacion por Internet se debe emplear un protocolo de transporte cifrado,  ssl: (sobre TCP) ó wss: (sobre WebSockets)")
		fmt.Println("       Los mensajes se publican en 'clean session' con qos 0 y con 'retained flag' para que en cada nueva conexión el subcriptor reciba un mensaje con el estado actual")
		fmt.Println("-mqpub2 <url>")
		fmt.Println("-mqpub3 <url>")
		fmt.Println("       Especifica segundo/tercer url de servidor/broker MQTT. Formato url:")
		fmt.Println("          protocolo://[usuario[:password]@]host.dominio.tld:puerto")
		fmt.Println("       Ignora base_topic si se pone al final de la url. Unicamente se condidera la especificada en -mqpub")
		fmt.Println()
		fmt.Println("-cbc")
		fmt.Println("       Check Broker Certificate. Habilita que se verifique el certificado presentado por el broker MQTT") 
		fmt.Println("       en los protocolos protegidos por TLS así como la cadena de certificación hasta el root certificate")
				
		//TODO --en Ingles ----
		//fmt.Println("Return via stdout events and current state of buttons and axes of DragonRise joystick board for IoT use")
		//fmt.Println("Read from device file /dev/input/js0 or other specified")
				
		os.Exit(0)
	}

	//For de inicailizacion de punteros a array de switches y conmutadores todas las tarjetas
	for i:=0; i<maxTarjetas; i++{
		tarjeta[i].Swt=switches[i][:maxSwt]
		tarjeta[i].Com=conmutadores[i][:maxCom]
	} 

	//  lo que hay detras de las opciones en la linea de comando son los dispositivos
	//  Se establece default si no especifica al menos un device 
	numTarjetas=1
	fDispositivo[0] = nombreFicheroDispositivoOmision
	fEstado[0]= statusFilePath + filepath.Base(fDispositivo[0]) + statusFileExt
	//  Se carga en los arrays los nombres de ficheros de dispositivos y de estado
	for i:=0; flag.Arg(i)!=""; i++ {
		fDispositivo[i] = flag.Arg(i)
		fEstado[i] = statusFilePath + filepath.Base(fDispositivo[i]) + statusFileExt
		numTarjetas = i + 1
	}


	
	if (mqpub!="") {
		if *pOpcionCbc == true {
			verificarCertificadoBroker=true
		} else{
			verificarCertificadoBroker=false
		}
		//TODO Inicializar todos los topics
		topic[0]=devuelveTopic(mqpub,fDispositivo[0]);
		
		inicioConexion(mqpub)
		if (mqpub2!="") {
			inicioConexion(mqpub2)
			if (mqpub3!=""){
				inicioConexion(mqpub3)
			}
		}	

	} else{
		fmt.Fprintf(os.Stderr, "\n%s", "No se ha especificado opción -mqpub. No se publicarán mensajes MQTT")
	}

	fmt.Fprintf(os.Stderr, "\nLista de interruptores y ejes/conmutadores con sus estados/valores actuales")

	for i:=0; i<numTarjetas; i++ {
		fmt.Fprintf(os.Stderr, "\n %d  %s   %s", i, fDispositivo[i], fEstado[i]) 
		reinicializaDragonrise(i)
		/*
		// se publica el estado inicial
		if (mqpub!=""){
			//Publicacion de estado tras apertura de la tarjeta desde el propio fichero de estado si se tiene mqpub
			estado, _ := ioutil.ReadFile(fEstado[numTarjeta])
			//TODO urg Retirar delay cuando se sincronice con goroutine que comunique la conexión. Evitar el delay
			//time.Sleep(1 * time.Second)
			fmt.Fprintf(os.Stderr, "\nPublicando %s   %s", topic[numTarjeta], string(estado))
			publicar(topic[numTarjeta], string(estado))
		}
		*/		
		go leerDevice(i)		
	}
	fmt.Fprintf(os.Stderr, "\n")
	
	select {
	}


/*

	// 	A la escucha de eventos ....

	var tipoSensor byte
	var posicion byte
	var valor int16

	for {
		leidos, _ := hDispositivo[numTarjeta].Read(buffer)
		if leidos!=8{
			fmt.Fprintf(os.Stderr, "\nError: Lectura de evento con menos de 8bytes")
		} 
		
		tipoSensor=buffer[6]&(0xFF^0x80)
		posicion = buffer[7]
		valor = int16(binary.LittleEndian.Uint16(buffer[4:6]))
	  
		er := tratarEvento(numTarjeta, true, tipoSensor, posicion ,valor)
		if (er==0 && mqpub!=""){
		//Publicacion de evento desde el propio fichero de estado si se tiene mqpub
			estado, _ := ioutil.ReadFile(fEstado[numTarjeta])
			publicar(topic[numTarjeta], string(estado))
		}
	}
*/	
}