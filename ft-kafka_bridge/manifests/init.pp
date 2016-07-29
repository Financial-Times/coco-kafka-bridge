class kafka_bridge {

  $configParameters = hiera('configParameters','')

  class { "go_service_profile" :
    service_module => $module_name,
    service_name => 'kafka-bridge',
    configParameters => $configParameters
  }

}
