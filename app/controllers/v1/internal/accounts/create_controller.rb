# typed: true
# frozen_string_literal: true

class V1::Internal::Accounts::CreateController < V1::Internal::ApplicationController
  include ControllerConcerns::InternalAuthenticatable
  include ControllerConcerns::Localizable
  include ControllerConcerns::V1::FormErrorable

  around_action :set_locale

  def call
    form = V1::AccountForm.new(
      atname: params[:atname],
      email: params[:email],
      password: params[:password],
      locale: "ja", # TODO: あとでユーザの言語を指定する
      time_zone: "Asia/Tokyo" # TODO: あとでユーザのタイムゾーンを指定する
    )

    if form.invalid?
      return response_form_errors(resource_class: V1::FormErrorResource, errors: form.errors)
    end

    result = CreateAccountUseCase.new.call(
      atname: form.atname.not_nil!,
      email: form.email.not_nil!,
      locale: form.locale.not_nil!,
      password: form.password.not_nil!,
      time_zone: form.time_zone.not_nil!
    )
    result.user.track_sign_in

    resource = V1::AccountResource.new(
      user: result.user,
      profile: result.profile,
      oauth_access_token: result.oauth_access_token
    )
    render(
      json: V1::AccountSerializer.new(resource),
      status: :created
    )
  end
end
