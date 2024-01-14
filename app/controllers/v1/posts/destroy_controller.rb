# typed: true
# frozen_string_literal: true

class V1::Posts::DestroyController < V1::ApplicationController
  include ControllerConcerns::PublicAuthenticatable
  include ControllerConcerns::V1::FormErrorable

  def call
    form = V1::PostDeleteForm.new(
      viewer: current_viewer!,
      target_post_id: params[:post_id]
    )

    if form.invalid?
      return response_form_errors(resource_class: V1::PostFormErrorResource, errors: form.errors)
    end

    DiscardPostUseCase.new.call(target_post: form.target_post.not_nil!)

    render(status: :no_content)
  end
end
