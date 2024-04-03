# typed: true

# DO NOT EDIT MANUALLY
# This is an autogenerated file for dynamic methods in `GeneratedPathHelpersModule`.
# Please instead update this file by running `bin/tapioca dsl GeneratedPathHelpersModule`.

module GeneratedPathHelpersModule
  include ::ActionDispatch::Routing::UrlFor
  include ::ActionDispatch::Routing::PolymorphicRoutes

  sig { params(args: T.untyped).returns(String) }
  def account_list_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def api_v1_profiles_me_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def api_v1_users_me_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def community_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def edit_password_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def email_confirmation_list_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def home_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def new_account_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def new_email_confirmation_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def new_post_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def notification_list_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def password_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def password_reset_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def post_list_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def post_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def post_stamp_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def preview_view_component_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def preview_view_components_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def privacy_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def profile_atom_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def profile_check_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def profile_follow_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def profile_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def profile_post_path(*args); end

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
  def search_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def settings_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def settings_profile_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def settings_user_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def sign_in_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def sign_out_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def sign_up_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def terms_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def v1_follow_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def v1_internal_account_list_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def v1_internal_email_confirmation_challenge_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def v1_internal_email_confirmation_list_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def v1_internal_email_confirmation_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def v1_internal_password_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def v1_internal_post_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def v1_internal_profile_post_list_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def v1_internal_session_list_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def v1_notification_list_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def v1_post_list_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def v1_post_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def v1_post_stamp_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def v1_profile_me_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def v1_profile_post_list_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def v1_suggested_profile_check_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def v1_suggested_profile_list_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def v1_timeline_path(*args); end

  sig { params(args: T.untyped).returns(String) }
  def v1_user_me_path(*args); end
end
