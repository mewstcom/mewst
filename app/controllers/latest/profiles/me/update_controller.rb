# typed: true
# frozen_string_literal: true

class Latest::Profiles::Me::UpdateController < Latest::ApplicationController
  include Latest::Authenticatable
  include Latest::FormErrorable

  def call
    form = Latest::ProfileForm.new(
      viewer: current_viewer!,
      atname: params[:atname],
      avatar_url: params[:avatar_url],
      description: params[:description],
      name: params[:name]
    )

    if form.invalid?
      return response_form_errors(resource_class: Latest::FormErrorResource, errors: form.errors)
    end

    result = UpdateProfileUseCase.new.call(
      viewer: form.viewer.not_nil!,
      atname: form.atname.not_nil!,
      avatar_url: form.avatar_url.not_nil!,
      description: form.description.not_nil!,
      name: form.name.not_nil!
    )

    profile_resource = Latest::ProfileResource.new(profile: result.profile, viewer: current_viewer!)
    render(
      json: {
        profile: Latest::ProfileSerializer.new(profile_resource).to_h
      },
      status: :ok
    )
  end
end
