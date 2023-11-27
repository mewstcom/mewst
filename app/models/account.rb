# typed: strict
# frozen_string_literal: true

class Account
  extend T::Sig

  sig { returns(T.nilable(Profile)) }
  attr_reader :profile

  sig { returns(T.nilable(User)) }
  attr_reader :user

  sig { returns(T.nilable(OauthAccessToken)) }
  attr_reader :oauth_access_token

  sig do
    params(
      atname: String,
      email: String,
      locale: String,
      password: String,
      current_time: ActiveSupport::TimeWithZone,
      time_zone: String
    ).void
  end
  def initialize(atname:, email:, locale:, password:, current_time: Time.current, time_zone: "UTC")
    @atname = atname
    @email = email
    @locale = locale
    @password = password
    @current_time = current_time
    @time_zone = time_zone
    @profile = T.let(nil, T.nilable(Profile))
    @user = T.let(nil, T.nilable(User))
    @oauth_access_token = T.let(nil, T.nilable(OauthAccessToken))
  end

  sig { void }
  def save!
    @profile = Profile.create!(
      profileable_type: ProfileableType::User.serialize,
      atname:,
      joined_at: current_time
    )

    @user = profile.not_nil!.create_user!(
      email:,
      password:,
      locale:,
      time_zone:,
      signed_up_at: current_time
    )

    actor = @profile.actors.create!(user: @user)

    @oauth_access_token = OauthAccessToken.find_or_create_for(
      application: OauthApplication.mewst_web,
      resource_owner: actor,
      scopes: ""
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

  sig { returns(String) }
  attr_reader :time_zone
  private :time_zone
end
