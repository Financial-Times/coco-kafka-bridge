class kafka_bridge {

  $binary_name = "kafka-bridge"
  $install_dir = "/usr/local/$binary_name"
  $binary_file = "$install_dir/$binary_name"
  $log_dir = "/var/log/apps"
  $config_file = "/etc/$binary_name.properties"

  class { 'common_pp_up': }
  class { "${module_name}::monitoring": }
  class { "${module_name}::supervisord": }

  file {
    $install_dir:
      mode    => "0664",
      ensure  => directory;

    $binary_file:
      ensure  => present,
      source  => "puppet:///modules/$module_name/$binary_name",
      mode    => "0755",
      require => File[$install_dir];

    $config_file:
      content => template("$module_name/kafka-bridge.properties.erb"),
      mode    => "0664";

    $log_dir:
      ensure  => directory,
      mode    => "0664"
  }

  exec { 'restart_kafka-bridge':
    command     => "supervisorctl restart $binary_name",
    path        => "/usr/bin:/usr/sbin:/bin",
    subscribe   => [
      File[$binary_file],
      File[$config_file],
      Class["${module_name}::supervisord"]
    ],
    refreshonly => true
  }
}
