# typed: strict
# frozen_string_literal: true

class Account
  extend T::Sig

  sig { returns(Profile) }
  attr_reader :profile

  sig { returns(User) }
  attr_reader :user

  sig { returns(OauthAccessToken) }
  attr_reader :oauth_access_token

  sig { params(atname: String, email: String, locale: String, password: String, current_time: ActiveSupport::TimeWithZone).void }
  def initialize(atname:, email:, locale:, password:, current_time: Time.current)
    @atname = atname
    @email = email
    @locale = locale
    @password = password
    @current_time = current_time
  end

  sig { void }
  def save!
    @profile = Profile.create!(
      actor_type: :user,
      atname:,
      joined_at: current_time
    )

    @user = profile.create_user!(
      email:,
      locale:,
      password:,
      signed_up_at: current_time
    )

    @oauth_access_token = OauthAccessToken.find_or_create_for(
      application: OauthApplication.mewst_web,
      resource_owner: profile,
      scopes: "",
      user:
    )

    nil
  end

  sig { returns(String) }
  attr_reader :atname
  private :atname

  sig { returns(String) }
  attr_reader :email
  private :email

  sig { returns(String) }
  attr_reader :locale
  private :locale

  sig { returns(String) }
  attr_reader :password
  private :password

  sig { returns(ActiveSupport::TimeWithZone) }
  attr_reader :current_time
  private :current_time
end
