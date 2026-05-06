# typed: true
# frozen_string_literal: true

class Api::V1::Profiles::Me::UpdateController < Api::V1::ApplicationController
  # include ControllerConcerns::PublicAuthenticatable
  # include ControllerConcerns::V1::FormErrorable
  #
  # def call
  #   form = V1::ProfileForm.new(
  #     viewer: current_viewer!,
  #     atname: params[:atname],
  #     avatar_url: params[:avatar_url],
  #     description: params[:description],
  #     name: params[:name]
  #   )
  #
  #   if form.invalid?
  #     return response_form_errors(resource_class: V1::FormErrorResource, errors: form.errors)
  #   end
  #
  #   result = UpdateProfileUseCase.new.call(
  #     viewer: form.viewer.not_nil!,
  #     atname: form.atname.not_nil!,
  #     avatar_url: form.avatar_url.not_nil!,
  #     description: form.description.not_nil!,
  #     name: form.name.not_nil!
  #   )
  #
  #   profile_resource = V1::ProfileResource.new(profile: result.profile, viewer: current_viewer!)
  #   render(
  #     json: {
  #       profile: V1::ProfileSerializer.new(profile_resource).to_h
  #     },
  #     status: :ok
  #   )
  # end
end
