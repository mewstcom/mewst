# typed: true
# frozen_string_literal: true

class V1::Stamps::DestroyController < V1::ApplicationController
  include PublicAuthenticatable
  include V1::FormErrorable

  def call
    form = V1::StampForm.new(
      viewer: current_viewer!,
      target_post_id: params[:post_id]
    )

    if form.invalid?
      return response_form_errors(resource_class: V1::StampFormErrorResource, errors: form.errors)
    end

    result = DeleteStampUseCase.new.call(
      viewer: form.viewer.not_nil!,
      target_post: form.target_post.not_nil!
    )

    resource = V1::PostResource.new(post: result.post, viewer: current_viewer!)
    render(
      json: V1::PostSerializer.new(resource),
      status: :ok
    )
  end
end
