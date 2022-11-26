# typed: true

# DO NOT EDIT MANUALLY
# This is an autogenerated file for dynamic methods in `GeneratedPathHelpersModule`.
# Please instead update this file by running `bin/tapioca dsl GeneratedPathHelpersModule`.

module GeneratedPathHelpersModule
  include ::ActionDispatch::Routing::UrlFor
  include ::ActionDispatch::Routing::PolymorphicRoutes

  sig { params(args: T.untyped).returns(String) }
  def accounts_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def home_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def internal_api_follow_toggle_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def internal_api_following_list_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def new_account_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def preview_view_component_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def preview_view_components_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def profile_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def rails_info_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def rails_info_properties_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def rails_info_routes_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def rails_mailers_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def root_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def sidekiq_web_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def sign_in_callback_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def sign_in_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def sign_out_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def sign_up_callback_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def sign_up_path(*args); end
end
