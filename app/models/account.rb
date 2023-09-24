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

  sig { params(atname: String, email: String, locale: String, password: String, current_time: ActiveSupport::TimeWithZone).void }
  def initialize(atname:, email:, locale:, password:, current_time: Time.current)
    @atname = atname
    @email = email
    @locale = locale
    @password = password
    @current_time = current_time
    @profile = T.let(nil, T.nilable(Profile))
    @user = T.let(nil, T.nilable(User))
    @oauth_access_token = T.let(nil, T.nilable(OauthAccessToken))
  end

  sig { void }
  def save!
    @profile = Profile.create!(
      profileable_type: :user,
      atname:,
      joined_at: current_time
    )

    @user = profile.not_nil!.create_user!(
      email:,
      locale:,
      password:,
      signed_up_at: current_time
    )

    actor = @profile.actors.create!(user: @user)

    @oauth_access_token = OauthAccessToken.find_or_create_for(
      application: OauthApplication.mewst_web,
      resource_owner: actor,
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
