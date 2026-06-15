import { ReactElement } from 'react';
import { Control, FieldValues, UseFormReturn } from 'react-hook-form';
import useFormPersist from './useFormPersist';

interface FormPersistProps<T extends FieldValues, TTransformedValues = T> {
  form: UseFormReturn<T, unknown, TTransformedValues>;
  formKey: string;
}
const isBrowser = () => typeof window !== 'undefined';

export default function FormPersist<
  T extends FieldValues,
  TTransformedValues = T,
>(props: FormPersistProps<T, TTransformedValues>): ReactElement {
  const { form, formKey } = props;
  useFormPersist(formKey, {
    // useFormPersist operates on string keys and is intentionally `any`-typed;
    // the form's concrete generics carry no extra safety here.
    control: form.control as Control<FieldValues>,
    setValue: form.setValue,
    storage: isBrowser() ? window.sessionStorage : undefined,
  });
  return <></>;
}
