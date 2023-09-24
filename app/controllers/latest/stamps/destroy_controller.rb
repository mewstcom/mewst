# typed: true
# frozen_string_literal: true

class Latest::Stamps::DestroyController < Latest::ApplicationController
  include Authenticatable
  include Latest::FormErrorable

  def call
    form = Latest::StampForm.new(
      viewer: current_viewer!,
      target_post_id: params[:post_id]
    )

    if form.invalid?
      return response_form_errors(resource_class: Latest::StampFormErrorResource, errors: form.errors)
    end

    result = DeleteStampUseCase.new.call(
      viewer: form.viewer.not_nil!,
      target_post: form.target_post.not_nil!
    )

    resource = Latest::PostResource.new(post: result.post, viewer: current_viewer!)
    render(
      json: Latest::PostSerializer.new(resource),
      status: :ok
    )
  end
end
