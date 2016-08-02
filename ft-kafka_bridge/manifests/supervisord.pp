class kafka_bridge::supervisord {

  $supervisord_init_file = "/etc/init.d/supervisord"
  $supervisord_config_file = "/etc/supervisord.conf"
  $supervisord_log_dir = "/var/log/supervisor"

  satellitesubscribe { 'gateway-epel':
    channel_name  => 'epel'
  }

package { 'python-pip':
    ensure    => 'installed',
    require   => Satellitesubscribe["gateway-epel"]
  }

package { 'supervisor':
    provider  => pip,
    ensure    => present,
    require   => Package['python-pip']
  }

file {
    $supervisord_init_file:
      mode      => "0755",
      content   => template("$module_name/supervisord.init.erb"),
      require   => Package['supervisor'],
      notify    => Service['supervisord'];

    $supervisord_config_file:
      mode      => "0664",
      content   => template("$module_name/supervisord.conf.erb"),
      require   => Package['supervisor'],
      notify    => Service['supervisord'];

    $supervisord_log_dir:
      ensure    => directory,
      mode      => "0664",
      require   => Package['supervisor']
  }

service { 'supervisord':
    ensure      => running,
    enable      => true,
    require     => File[$supervisord_log_dir]
  }
}