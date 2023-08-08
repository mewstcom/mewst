# typed: strict
# frozen_string_literal: true

class CreateAccountService::Input < T::Struct
  extend T::Sig

  const :atname, String
  const :email, String
  const :locale, String
  const :password, String

  sig { params(form: Internal::AccountForm).returns(CreateAccountService::Input) }
  def self.from_internal_form(form:)
    new(
      atname: form.atname,
      email: form.email,
      locale: form.locale,
      password: form.password
    )
  end
end
