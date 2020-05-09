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
)
// constantes
const(
	versionFecha = "v018 - 9 mayo 2020" 
	bufferSize =8 //numero de bytes de buffer de lectura
	statusFileNameDefault = "js0.dat" 
	statusFilePath = "/var/lib/dragonrise/"   
	maxSwt = 12    //número máximo de interruptores
	maxCom = 7     //número máximo de ejes-conmutadores 
)
//definicion de tipos
type evento struct{
	TipoSensor byte 	`json:"type"`
	NumSensor byte     `json:"sensor"`
	ValorSensor int16	`json:"value"`
}

type dragonrise struct{
	Tiempo int64 		`json:"time"`
	Evento evento   	`json:"event"`// último evento registrado
	Swt []int16   		`json:"switches"`// estado actual de interruptores
	Com []int16   		`json:"axes"`// estado actual de ejes/conmutadores
}

//variables globales
var (
	switches [maxSwt]int16     	// array de interruptores
	conmutadores [maxCom]int16 	// array de ejes/conmutadores
	tarjeta dragonrise			// estructura de ultimo evento y estados	
	fs *os.File  				//handle de fichero de estado
)

// funcion callback  handler OnConnect  Se llama en la conexion inicial y el las posteriores reconexiones
func ch (client mqtt.Client) {
	fmt.Fprintf(os.Stderr, "Conectado broker: \n")
}

func getMacAddr() ([]string, error) {
    ifas, err := net.Interfaces()
    if err != nil {
        return nil, err
    }
	// fmt.Println(ifas)   //para depuracion
	// llena un arrau de strings con cada interfaz que tenga mac address
    var as []string
    for _, ifa := range ifas {
        a := ifa.HardwareAddr.String()
        if a != "" {
            as = append(as, a)
        }
    }
    return as, nil
}

func check(e error) {
    if e != nil {
        panic(e)
    }
}

/*  
Registra eventos sinteticos (de conocimiento estado inicial) y reales 
en la struct de estado dragonrise, salvo que tipoSensor == 0
Saca por stdout los valores de la struct, salvo eventoReal==false 
*/
func tratarEvento (eventoReal bool, tipoSensor byte, numSensor byte, valorSensor int16) (error int){
	switch tipoSensor{
		//no registra estado en struct. Se asegura que numSensor y valorSensor sean 0
		case 0:	
			numSensor=0
			valorSensor=0
		// en estos dos casos SI registra estado en struct
		case 1:
			tarjeta.Swt[numSensor] = valorSensor
		case 2:
			//Descarta eventos espureos de ejes distitos a 0 o 1 ¡¡cuidadito con el algebra de Boole!!
			if !(numSensor==0 || numSensor==1){
				return 1
			}
			valorSensor=  valorSensor/32767   //si es tipo eje se normaliza el valor (-1 0 +1)
			tarjeta.Com[numSensor]= valorSensor;
	}
	tarjeta.Evento.TipoSensor = tipoSensor
	tarjeta.Evento.NumSensor = numSensor
	tarjeta.Evento.ValorSensor = valorSensor 
	
	// si eventoReal == false --> evento sintetico inicial
	if eventoReal == false {
		tarjeta.Tiempo = 0 // --> evento sintetico inicial
	} else {
		tarjeta.Tiempo = time.Now().Unix()  // --> evento real
	}
	if tipoSensor == 0 || eventoReal == true{
		salida, _ := json.Marshal(&tarjeta)
		fmt.Println(string(salida))	
		//TODO: Tratar errores
		fs.Truncate(0)
		fs.Seek(0,0)
		fs.Write([]byte(salida))
		fs.Sync()
	}
	return 0
}

func main(){
	fmt.Fprintf(os.Stderr,"stderr: dragonrise %s  autor:junav (junav2@hotmail.com)\n", versionFecha)

	pOpcionH := flag.Bool("h", false, "Muestra más información de ayuda")
	var mqpub string
	flag.StringVar(&mqpub, "mqpub", "", "URL MQTT Broker y topic de publicacion de mensajes mqttprotocol://host.dominio:puerto/base_topic")
	pOpcionCbc := flag.Bool("cbc", false, "Check Broker Certificate. Si no se pone esta opción el certificado presentado por el MQTT broker no será comprobado")
	
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
		fmt.Println("       Ejemplos:")
		fmt.Println("       -mqpub=tcp://host.dominio.dom:1883/base_topic")
		fmt.Println("       -mqpub=ssl://[usuario[:password]@]host.dominio.dom:8883/base_topic")
		fmt.Println("       -mqpub=ws://host.dominio.dom:80/base_topic")
		fmt.Println("       -mqpub=wss://[usuario[:password]@]host.dominio.dom:443/base_topic")
		fmt.Println("       Para enviar credenciales de autenticacion por Internet se debe emplear un protocolo de transporte cifrado,  ssl: (sobre TCP) ó wss: (sobre WebSockets)")
		fmt.Println("       Los mensajes se publican en 'clean session' con qos 0 y con 'retained flag' para que en cada nueva conexión el subcriptor reciba un mensaje con el estado actual")
		fmt.Println()
		fmt.Println("-cbc")
		fmt.Println("       Check Broker Certificate. Habilita que se verifique el certificado presentado por el broker MQTT") 
		fmt.Println("       en los protocolos protegidos por TLS así como la cadena de certificación hasta el root certificate")
				
		//TODO --en Ingles ----
		//fmt.Println("Return via stdout events and current state of buttons and axes of DragonRise joystick board for IoT use")
		//fmt.Println("Read from device file /dev/input/js0 or other specified")
				
		os.Exit(0)
	}
	urlMqttBroker:=""
	dragonriseEventTopic:= ""
	usuario:=string("")
	password:=string("") 
	
	if (mqpub!="") {
		uri, _ := url.Parse(mqpub)
		urlMqttBroker = fmt.Sprintf("%s://%s", uri.Scheme, uri.Host)
		pUsuariopassword:=uri.User
		if pUsuariopassword != nil {
			usuario=(*pUsuariopassword).Username()
			password, _=(*pUsuariopassword).Password()
		}
		baseTopic := uri.Path[1:len(uri.Path)] // retira el primer slash
		//TODO: asegurar NO slash final en base topic
		dragonriseEventTopic = baseTopic+"/"+filepath.Base(flag.Arg(0))+"/event"
		fmt.Fprintf(os.Stderr,"URL broker MQTT: %s\n", urlMqttBroker)
		fmt.Fprintf(os.Stderr,"Usuario : contraseña --> %s : %s\n", usuario, password)
		fmt.Fprintf(os.Stderr,"Topic Eventos(publicación): %s\n", dragonriseEventTopic)
	} else{
		fmt.Fprintf(os.Stderr, "%s\n", "No se ha especificado opción -mqpub. No se publicarán mensajes MQTT")
	}
	
	buffer:= make([]byte, bufferSize)
	var device string
	var statusFileName string
	
	// lo que hay detras de las opciones en la linea de comando ...
	if flag.Arg(0)=="" {
        device="/dev/input/js0"
		statusFileName=statusFileNameDefault	
    } else{
		device=flag.Arg(0);
		statusFileName=filepath.Base(flag.Arg(0))+ ".dat"	
	}

	leidos:=0
	
	var tipoSensor byte
	var posicion byte
	var valor int16
	
	tarjeta.Swt=switches[:maxSwt]
	tarjeta.Com=conmutadores[:maxCom]
	
	fmt.Fprintf(os.Stderr, "Abriendo %s\n", device)
	f, err := os.Open(device)
	check(err)
	defer f.Close()
	
	fmt.Fprintf(os.Stderr, "Abriendo %s\n", statusFilePath + statusFileName)	
	fs, err = os.OpenFile(statusFilePath + statusFileName, os.O_RDWR, 0755) //fs está declarada a nivel global para que pueda acceder la funcion tratarEvento()
	if os.IsPermission(err){
		check(err)
	}
	if os.IsNotExist(err){
		fmt.Fprintf(os.Stderr, "%s\n", err)
		err:=os.Mkdir(statusFilePath, 0744)
		fmt.Fprintf(os.Stderr, "%s\n", err)
		fs, err = os.OpenFile(statusFilePath + statusFileName, os.O_RDWR|os.O_CREATE, 0755)
		check(err)
	}
	defer fs.Close()
		
	fmt.Fprintf(os.Stderr, "List of switches and axes and initial/current values\n")
	
	/*
	El controlador joystick /dev/input/js0 tras la apertura del fichero de dispositivo devuelve en orden "eventos sintéticos" (no reales)
	de cada interruptor o eje/conmutador para informar de su estado actual. En el caso de dragonrise primero genera 
	12 eventos para los 12 interuptores y despues 7 eventos para eventos de los	ejes/conmutadores.
	Tras eso las lecturas al dispositivos estaran bloqueadas en espera de eventos reales.
	*/

    for nSwt:=0 ; nSwt<maxSwt; nSwt++ {
		leidos, err = f.Read(buffer)
		check(err)
		if leidos!=8{
		// error de inicializacion
			fmt.Fprintf(os.Stderr, "Error de inicializacion switches\n")
			os.Exit(0)
		}
		
		if buffer[6]!=0x81 {
			// error de inicializacion
			fmt.Fprintf(os.Stderr, "Error de inicializacion switches\n")
			os.Exit(0)
		}
		
		tipoSensor=buffer[6]&(0xFF^0x80)
		posicion = buffer[7]
		valor = int16(binary.LittleEndian.Uint16(buffer[4:6]))
		    
		tratarEvento(false, tipoSensor, posicion ,valor)
	}
		
	for nCom:=0 ; nCom<maxCom; nCom++ {
		leidos, err = f.Read(buffer)
		check(err)
		if leidos!=8{
		// error de inicializacion
			fmt.Fprintf(os.Stderr, "Error de inicializacion switches\n")
			os.Exit(0)
		}
		
		if buffer[6]!=0x82 {
			// error de inicializacion
			fmt.Fprintf(os.Stderr, "Error de inicializacion switches\n")
			os.Exit(0)
		}
		
		tipoSensor=buffer[6]&(0xFF^0x80)
		posicion = buffer[7]
		valor = int16(binary.LittleEndian.Uint16(buffer[4:6]))
		    
		tratarEvento(false, tipoSensor, posicion ,valor)
	}
	
	// terminada la inicializacion. Se solicita la salida del estado inicial  evento con eventoreal=false y tipo sensor=0 
	tratarEvento(false,0,0,0)

	//Conexion con broker mosquitto
	var c mqtt.Client
	if (urlMqttBroker!=""){ 
	
		//create a ClientOptions	
		opts := mqtt.NewClientOptions()
	
		// se ajustan valores de configuracion de la conexion de este cliente
		opts.AddBroker(urlMqttBroker)
		//opts.SetCleanSession(true)  //true es el default
		//opts.SetProtocolVersion(4)  //version de protocolo 4=MQTT 3.1.1 (default)    3=MQTT 3.1
		opts.SetUsername(usuario)
		opts.SetPassword(password)
		
		// se pone la primera mac address como clientID
		// se obtienen todas las MACs del dispositivo
		macs, err:=getMacAddr() 
		if err != nil {
			panic(err)
		}
		//Se pone como ClientID la primera MAC seguido del fichero de dispositivo. No pueden conectarse al broker dos cliente con mismo ClientID
		//TODO Verificar que longitud ClientUD no supera max del estandar MQTT  a client id must be no longer than 23 characters.
		opts.SetClientID(macs[0]+"-"+ filepath.Base(flag.Arg(0))) 
		opts.SetKeepAlive(2 * time.Second)
		opts.SetPingTimeout(1 * time.Second)
		opts.SetOnConnectHandler (ch)
		// Salvo que se especifique opcion -cbc, se ajusta la configuracion TLS para que no se verifique el certificado que presente el broker
		if *pOpcionCbc == false {
			//tlsconfig := NewTLSConfig()
			tlsConfig := &tls.Config{InsecureSkipVerify: true, ClientAuth: tls.NoClientCert}
			opts.SetTLSConfig(tlsConfig) 
		}
		//Crea una estructura con los datos de conexión del cliente
		c = mqtt.NewClient(opts)
		if token := c.Connect(); token.Wait() && token.Error() != nil {
		//TODO sustituir por mensaje en stderr
			fmt.Fprintf(os.Stderr,"Error conexion con broker MQTT:%s\n", token.Error())
		}
	}

	// 	A la escucha de eventos ....
	for {
		leidos, err := f.Read(buffer)
		check(err)
		if leidos!=8{
			fmt.Fprintf(os.Stderr, "Error: Lectura de evento con menos de 8bytes\n")
		} 
		
		tipoSensor=buffer[6]&(0xFF^0x80)
		posicion = buffer[7]
		valor = int16(binary.LittleEndian.Uint16(buffer[4:6]))
		    
		tratarEvento(true, tipoSensor, posicion ,valor)
 		
		//Publicacion de evento desde el propio fichero de estado si se tiene mqpub
		if (mqpub!=""){
			estado, _ := ioutil.ReadFile(statusFilePath + statusFileName)
		
			// Publish will publish a message with the specified QoS and content
			// to the specified topic.
			// Returns a token to track delivery of the message to the broker
			// Publish(topic string, qos byte, retained bool, payload interface{}) Token

			// Se pone el flag de "retained" del ultimo mensaje para que al conectar el subcriptor reciba el estado actual
			token := c.Publish(dragonriseEventTopic, 0, true, string(estado))
			token.Wait()
		}
	}
}