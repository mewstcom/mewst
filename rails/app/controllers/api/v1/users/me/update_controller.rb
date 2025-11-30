# typed: true
# frozen_string_literal: true

class Api::V1::Users::Me::UpdateController < Api::V1::ApplicationController
  # include ControllerConcerns::PublicAuthenticatable
  # include ControllerConcerns::V1::FormErrorable
  #
  # def call
  #   form = V1::UserForm.new(
  #     locale: params[:locale],
  #     time_zone: params[:time_zone]
  #   )
  #
  #   if form.invalid?
  #     return response_form_errors(resource_class: V1::FormErrorResource, errors: form.errors)
  #   end
  #
  #   result = UpdateUserUseCase.new.call(
  #     user: current_viewer!.user.not_nil!,
  #     locale: form.locale.not_nil!,
  #     time_zone: form.time_zone.not_nil!
  #   )
  #
  #   user_resource = V1::UserResource.new(user: result.user)
  #   render(
  #     json: {
  #       user: V1::UserSerializer.new(user_resource).to_h
  #     },
  #     status: :ok
  #   )
  # end
end
