import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpHandler;
import com.sun.net.httpserver.HttpServer;

import javax.management.InstanceAlreadyExistsException;
import javax.management.MBeanRegistrationException;
import javax.management.MBeanServer;
import javax.management.MalformedObjectNameException;
import javax.management.NotCompliantMBeanException;
import javax.management.ObjectName;
import java.io.IOException;
import java.io.OutputStream;
import java.lang.management.ManagementFactory;
import java.net.InetSocketAddress;
import java.util.Hashtable;
import java.util.Random;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicInteger;

class SimpleApp {
    public interface SampleMBean {

        Integer getShouldBe100();

        Double getShouldBe200();

        Long getShouldBe1337();

        Float getShouldBe1_1();

        int getShouldBeCounter();

        int getShouldBeSlowCounter();

        int incrementAndGet();

        int getSpecialCounter();

        int incrementSpecialCounter();

        int getIncrementCounter();
    }

    public static class Sample implements SampleMBean {

        private final AtomicInteger counter = new AtomicInteger(0);
        private final AtomicInteger incrementCounter = new AtomicInteger(0);
        private final AtomicInteger specialCounter = new AtomicInteger(0);
        private final Random random = new Random();

        @Override
        public Integer getShouldBe100() {
            return 100;
        }

        @Override
        public Double getShouldBe200() {
            return 200.0;
        }

        @Override
        public Long getShouldBe1337() {
            return 1337L;
        }

        @Override
        public Float getShouldBe1_1() {
            return 1.1F;
        }

        @Override
        public int getShouldBeCounter() {
            return this.counter.get();
        }

        @Override
        public int getShouldBeSlowCounter() {
            try {
                final int seconds = this.random.nextInt(15) + 20;
                Thread.sleep(seconds * 1000);
            } catch (InterruptedException e) {
                e.printStackTrace();
            }
            return this.counter.get();
        }

        @Override
        public int incrementAndGet() {
            return this.counter.incrementAndGet();
        }

        @Override
        public int getSpecialCounter() {
            return this.specialCounter.get();
        }

        @Override
        public int incrementSpecialCounter() {
            return this.specialCounter.incrementAndGet();
        }

        @Override
        public int getIncrementCounter() {
            return this.incrementCounter.incrementAndGet();
        }
    }

    public static void main(String[] args) throws Exception {
        System.out.println("Starting sample app...");
        try {
            final Hashtable<String, String> pairs = new Hashtable<>();
            pairs.put("name", "default");
            pairs.put("type", "simple");
            final SampleMBean sample = new Sample();
            final Thread daemonThread = getThread(pairs, sample);
            daemonThread.start();
            final HttpServer server = HttpServer.create(new InetSocketAddress(8000), 0);
            server.createContext("/test", new TestHandler(sample));
            server.createContext("/increment", new IncHandler(sample));
            server.setExecutor(null);
            server.start();
        } catch (MalformedObjectNameException | InstanceAlreadyExistsException |
                 MBeanRegistrationException | NotCompliantMBeanException | IOException e) {
            throw new RuntimeException(e);
        }
    }

    private static class IncHandler implements HttpHandler {
        private final SampleMBean sample;

        public IncHandler(final SampleMBean sample) {
            this.sample = sample;
        }

        @Override
        public void handle(HttpExchange t) throws IOException {
            this.sample.incrementSpecialCounter();
            final String response = "This is the current count of " + this.sample.getShouldBeCounter() + "\n";
            t.sendResponseHeaders(200, response.length());
            final OutputStream os = t.getResponseBody();
            os.write(response.getBytes());
            os.close();
        }
    }

    private static class TestHandler implements HttpHandler {
        private final SampleMBean sample;

        public TestHandler(final SampleMBean sample) {
            this.sample = sample;
        }

        @Override
        public void handle(HttpExchange t) throws IOException {
            final String response = "This is the current count of " + this.sample.getShouldBeCounter() + "\n";
            t.sendResponseHeaders(200, response.length());
            final OutputStream os = t.getResponseBody();
            os.write(response.getBytes());
            os.close();
        }
    }

    private static Thread getThread(final Hashtable<String, String> pairs, final SampleMBean sample)
            throws MalformedObjectNameException, InstanceAlreadyExistsException, MBeanRegistrationException, NotCompliantMBeanException {
        final ObjectName objectName = new ObjectName("dd.test.sample", pairs);
        final MBeanServer server = ManagementFactory.getPlatformMBeanServer();
        server.registerMBean(sample, objectName);
        final Thread daemonThread = new Thread(new Runnable() {
            @Override
            public void run() {
                while (sample.incrementAndGet() > 0) {
                    try {
                        Thread.sleep(TimeUnit.SECONDS.toSeconds(5));
                    } catch (InterruptedException e) {
                        throw new RuntimeException(e);
                    }
                }
            }
        });
        daemonThread.setDaemon(true);
        System.out.println("Daemon thread started");
        return daemonThread;
    }
}
